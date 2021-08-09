package mockgen

import (
	"bytes"
	"flag"
	"fmt"
	"github.com/golang/mock/mockgen/model"
	"go/token"
	"log"
	"path"
	"sort"
	"strconv"
	"strings"
	"unicode"

	toolsimports "golang.org/x/tools/imports"
)

//                                                                                                                       
//                                                                                                                       
//                                                                                                                       
//        ***************************     ***************************         *********      ************************    
//      *****************************    ******************************      *********      *************************    
//     *****************************     *******************************     *********     *************************     
//    *********                         *********                *******    *********     *********                      
//    ********                          *********               ********    *********     ********                       
//   ********     ******************   *********  *********************    *********     *********                       
//   ********     *****************    *********  ********************     *********     ********                        
//  ********      ****************    *********     ****************      *********     *********                        
//  ********                          *********      ********             *********     ********                         
// *********                         *********         ******            *********     *********                         
// ******************************    *********          *******          *********     *************************         
//  ****************************    *********            *******        *********      *************************         
//    **************************    *********              ******       *********         *********************          
//                                                                                                                  
//

const (
	gomockImportPath = "github.com/golang/mock/gomock"
)

var (
	selfPackage     = flag.String("self_package", "", "The full package import path for the generated code. The purpose of this flag is to prevent import cycles in the generated code by trying to include its own package. This can happen if the mock's package is set to one of its inputs (usually the main one) and the output is stdio so mockgen cannot detect the final output package. Setting this flag will then tell mockgen which import to exclude.")
	writePkgComment = flag.Bool("write_package_comment", true, "Writes package documentation comment (godoc) if true.")
)

// Generator to generate mock file
type Generator struct {
	buf                       bytes.Buffer
	indent                    string
	mockNames                 map[string]string // may be empty
	filename                  string            // may be empty
	destination               string            // may be empty
	srcPackage, srcInterfaces string            // may be empty
	copyrightHeader           string

	packageMap map[string]string // map from import path to package name
}
// GeneratorOption change the Generator Option
type GeneratorOption func(g *Generator)

// NewGenerator return a Generator Point
func NewGenerator(generatorOptions ...GeneratorOption) *Generator {
	g := new(Generator)
	for _, opt := range generatorOptions {
		opt(g)
	}
	return g
}
// OptGeneratorFilename change the Generator Prop of filename
func OptGeneratorFilename(filename string) GeneratorOption {
	return func(g *Generator) {
		g.filename = filename
	}
}

func (g *Generator) p(format string, args ...interface{}) {
	fmt.Fprintf(&g.buf, g.indent+format+"\n", args...)
}

func (g *Generator) in() {
	g.indent += "\t"
}

func (g *Generator) out() {
	if len(g.indent) > 0 {
		g.indent = g.indent[0 : len(g.indent)-1]
	}
}

// sanitize cleans up a string to make a suitable package name.
func sanitize(s string) string {
	t := ""
	for _, r := range s {
		if t == "" {
			if unicode.IsLetter(r) || r == '_' {
				t += string(r)
				continue
			}
		} else {
			if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' {
				t += string(r)
				continue
			}
		}
		t += "_"
	}
	if t == "_" {
		t = "x"
	}
	return t
}

// Generate gen the mock file
func (g *Generator) Generate(pkg *model.Package, outputPkgName string, outputPackagePath string) error {
	if outputPkgName != pkg.Name && *selfPackage == "" {
		// reset outputPackagePath if it's not passed in through -self_package
		outputPackagePath = ""
	}

	if g.copyrightHeader != "" {
		lines := strings.Split(g.copyrightHeader, "\n")
		for _, line := range lines {
			g.p("// %s", line)
		}
		g.p("")
	}

	g.p("// Code generated by MockGen. DO NOT EDIT.")
	if g.filename != "" {
		g.p("// Source: %v", g.filename)
	} else {
		g.p("// Source: %v (interfaces: %v)", g.srcPackage, g.srcInterfaces)
	}
	g.p("")

	// Get all required imports, and generate unique names for them all.
	im := pkg.Imports()
	im[gomockImportPath] = true

	// Only import reflect if it's used. We only use reflect in mocked methods
	// so only import if any of the mocked interfaces have methods.
	for _, intf := range pkg.Interfaces {
		if len(intf.Methods) > 0 {
			im["reflect"] = true
			break
		}
	}

	// Sort keys to make import alias generation predictable
	sortedPaths := make([]string, len(im))
	x := 0
	for pth := range im {
		sortedPaths[x] = pth
		x++
	}
	sort.Strings(sortedPaths)

	packagesName := createPackageMap(sortedPaths)

	g.packageMap = make(map[string]string, len(im))
	localNames := make(map[string]bool, len(im))
	for _, pth := range sortedPaths {
		base, ok := packagesName[pth]
		if !ok {
			base = sanitize(path.Base(pth))
		}

		// Local names for an imported package can usually be the basename of the import path.
		// A couple of situations don't permit that, such as duplicate local names
		// (e.g. importing "html/template" and "text/template"), or where the basename is
		// a keyword (e.g. "foo/case").
		// try base0, base1, ...
		pkgName := base
		i := 0
		for localNames[pkgName] || token.Lookup(pkgName).IsKeyword() {
			pkgName = base + strconv.Itoa(i)
			i++
		}

		// Avoid importing package if source pkg == output pkg
		if pth == pkg.PkgPath && outputPackagePath == pkg.PkgPath {
			continue
		}

		g.packageMap[pth] = pkgName
		localNames[pkgName] = true
	}

	if *writePkgComment {
		g.p("// Package %v is a generated GoMock package.", outputPkgName)
	}
	g.p("package %v", outputPkgName)
	g.p("")
	g.p("import (")
	g.in()
	for pkgPath, pkgName := range g.packageMap {
		if pkgPath == outputPackagePath {
			continue
		}
		g.p("%v %q", pkgName, pkgPath)
	}
	for _, pkgPath := range pkg.DotImports {
		g.p(". %q", pkgPath)
	}
	g.out()
	g.p(")")

	for _, intf := range pkg.Interfaces {
		if err := g.GenerateMockInterface(intf, outputPackagePath); err != nil {
			return err
		}
	}

	return nil
}

// The name of the mock type to use for the given interface identifier.
func (g *Generator) mockName(typeName string) string {
	if mockName, ok := g.mockNames[typeName]; ok {
		return mockName
	}

	return "Mock" + typeName
}

// GenerateMockInterface ...
func (g *Generator) GenerateMockInterface(intf *model.Interface, outputPackagePath string) error {
	mockType := g.mockName(intf.Name)

	g.p("")
	g.p("// %v is a mock of %v interface.", mockType, intf.Name)
	g.p("type %v struct {", mockType)
	g.in()
	g.p("ctrl     *gomock.Controller")
	g.p("recorder *%vMockRecorder", mockType)
	g.out()
	g.p("}")
	g.p("")

	g.p("// %vMockRecorder is the mock recorder for %v.", mockType, mockType)
	g.p("type %vMockRecorder struct {", mockType)
	g.in()
	g.p("mock *%v", mockType)
	g.out()
	g.p("}")
	g.p("")

	g.p("// New%v creates a new mock instance.", mockType)
	g.p("func New%v(ctrl *gomock.Controller) *%v {", mockType, mockType)
	g.in()
	g.p("mock := &%v{ctrl: ctrl}", mockType)
	g.p("mock.recorder = &%vMockRecorder{mock}", mockType)
	g.p("return mock")
	g.out()
	g.p("}")
	g.p("")

	// XXX: possible name collision here if someone has EXPECT in their interface.
	g.p("// EXPECT returns an object that allows the caller to indicate expected use.")
	g.p("func (m *%v) EXPECT() *%vMockRecorder {", mockType, mockType)
	g.in()
	g.p("return m.recorder")
	g.out()
	g.p("}")

	g.GenerateMockMethods(mockType, intf, outputPackagePath)

	return nil
}

type byMethodName []*model.Method

func (b byMethodName) Len() int           { return len(b) }
func (b byMethodName) Swap(i, j int)      { b[i], b[j] = b[j], b[i] }
func (b byMethodName) Less(i, j int) bool { return b[i].Name < b[j].Name }

// GenerateMockMethods ...
func (g *Generator) GenerateMockMethods(mockType string, intf *model.Interface, pkgOverride string) {
	sort.Sort(byMethodName(intf.Methods))
	for _, m := range intf.Methods {
		g.p("")
		_ = g.GenerateMockMethod(mockType, m, pkgOverride)
		g.p("")
		_ = g.GenerateMockRecorderMethod(mockType, m)
	}
}

func makeArgString(argNames, argTypes []string) string {
	args := make([]string, len(argNames))
	for i, name := range argNames {
		// specify the type only once for consecutive args of the same type
		if i+1 < len(argTypes) && argTypes[i] == argTypes[i+1] {
			args[i] = name
		} else {
			args[i] = name + " " + argTypes[i]
		}
	}
	return strings.Join(args, ", ")
}

// GenerateMockMethod generates a mock method implementation.
// If non-empty, pkgOverride is the package in which unqualified types reside.
func (g *Generator) GenerateMockMethod(mockType string, m *model.Method, pkgOverride string) error {
	argNames := g.getArgNames(m)
	argTypes := g.getArgTypes(m, pkgOverride)
	argString := makeArgString(argNames, argTypes)

	rets := make([]string, len(m.Out))
	for i, p := range m.Out {
		rets[i] = p.Type.String(g.packageMap, pkgOverride)
	}
	retString := strings.Join(rets, ", ")
	if len(rets) > 1 {
		retString = "(" + retString + ")"
	}
	if retString != "" {
		retString = " " + retString
	}

	ia := newIdentifierAllocator(argNames)
	idRecv := ia.allocateIdentifier("m")

	g.p("// %v mocks base method.", m.Name)
	g.p("func (%v *%v) %v(%v)%v {", idRecv, mockType, m.Name, argString, retString)
	g.in()
	g.p("%s.ctrl.T.Helper()", idRecv)

	var callArgs string
	if m.Variadic == nil {
		if len(argNames) > 0 {
			callArgs = ", " + strings.Join(argNames, ", ")
		}
	} else {
		// Non-trivial. The generated code must build a []interface{},
		// but the variadic argument may be any type.
		idVarArgs := ia.allocateIdentifier("varargs")
		idVArg := ia.allocateIdentifier("a")
		g.p("%s := []interface{}{%s}", idVarArgs, strings.Join(argNames[:len(argNames)-1], ", "))
		g.p("for _, %s := range %s {", idVArg, argNames[len(argNames)-1])
		g.in()
		g.p("%s = append(%s, %s)", idVarArgs, idVarArgs, idVArg)
		g.out()
		g.p("}")
		callArgs = ", " + idVarArgs + "..."
	}
	if len(m.Out) == 0 {
		g.p(`%v.ctrl.Call(%v, %q%v)`, idRecv, idRecv, m.Name, callArgs)
	} else {
		idRet := ia.allocateIdentifier("ret")
		g.p(`%v := %v.ctrl.Call(%v, %q%v)`, idRet, idRecv, idRecv, m.Name, callArgs)

		// Go does not allow "naked" type assertions on nil values, so we use the two-value form here.
		// The value of that is either (x.(T), true) or (Z, false), where Z is the zero value for T.
		// Happily, this coincides with the semantics we want here.
		retNames := make([]string, len(rets))
		for i, t := range rets {
			retNames[i] = ia.allocateIdentifier(fmt.Sprintf("ret%d", i))
			g.p("%s, _ := %s[%d].(%s)", retNames[i], idRet, i, t)
		}
		g.p("return " + strings.Join(retNames, ", "))
	}

	g.out()
	g.p("}")
	return nil
}

// GenerateMockRecorderMethod ...
func (g *Generator) GenerateMockRecorderMethod(mockType string, m *model.Method) error {
	argNames := g.getArgNames(m)

	var argString string
	if m.Variadic == nil {
		argString = strings.Join(argNames, ", ")
	} else {
		argString = strings.Join(argNames[:len(argNames)-1], ", ")
	}
	if argString != "" {
		argString += " interface{}"
	}

	if m.Variadic != nil {
		if argString != "" {
			argString += ", "
		}
		argString += fmt.Sprintf("%s ...interface{}", argNames[len(argNames)-1])
	}

	ia := newIdentifierAllocator(argNames)
	idRecv := ia.allocateIdentifier("mr")

	g.p("// %v indicates an expected call of %v.", m.Name, m.Name)
	g.p("func (%s *%vMockRecorder) %v(%v) *gomock.Call {", idRecv, mockType, m.Name, argString)
	g.in()
	g.p("%s.mock.ctrl.T.Helper()", idRecv)

	var callArgs string
	if m.Variadic == nil {
		if len(argNames) > 0 {
			callArgs = ", " + strings.Join(argNames, ", ")
		}
	} else {
		if len(argNames) == 1 {
			// Easy: just use ... to push the arguments through.
			callArgs = ", " + argNames[0] + "..."
		} else {
			// Hard: create a temporary slice.
			idVarArgs := ia.allocateIdentifier("varargs")
			g.p("%s := append([]interface{}{%s}, %s...)",
				idVarArgs,
				strings.Join(argNames[:len(argNames)-1], ", "),
				argNames[len(argNames)-1])
			callArgs = ", " + idVarArgs + "..."
		}
	}
	g.p(`return %s.mock.ctrl.RecordCallWithMethodType(%s.mock, "%s", reflect.TypeOf((*%s)(nil).%s)%s)`, idRecv, idRecv, m.Name, mockType, m.Name, callArgs)

	g.out()
	g.p("}")
	return nil
}

func (g *Generator) getArgNames(m *model.Method) []string {
	argNames := make([]string, len(m.In))
	for i, p := range m.In {
		name := p.Name
		if name == "" || name == "_" {
			name = fmt.Sprintf("arg%d", i)
		}
		argNames[i] = name
	}
	if m.Variadic != nil {
		name := m.Variadic.Name
		if name == "" {
			name = fmt.Sprintf("arg%d", len(m.In))
		}
		argNames = append(argNames, name)
	}
	return argNames
}

func (g *Generator) getArgTypes(m *model.Method, pkgOverride string) []string {
	argTypes := make([]string, len(m.In))
	for i, p := range m.In {
		argTypes[i] = p.Type.String(g.packageMap, pkgOverride)
	}
	if m.Variadic != nil {
		argTypes = append(argTypes, "..."+m.Variadic.Type.String(g.packageMap, pkgOverride))
	}
	return argTypes
}

type identifierAllocator map[string]struct{}

func newIdentifierAllocator(taken []string) identifierAllocator {
	a := make(identifierAllocator, len(taken))
	for _, s := range taken {
		a[s] = struct{}{}
	}
	return a
}

func (o identifierAllocator) allocateIdentifier(want string) string {
	id := want
	for i := 2; ; i++ {
		if _, ok := o[id]; !ok {
			o[id] = struct{}{}
			return id
		}
		id = want + "_" + strconv.Itoa(i)
	}
}

// Output returns the Generator's output, formatted in the standard Go style.
func (g *Generator) Output() []byte {
	src, err := toolsimports.Process(g.destination, g.buf.Bytes(), nil)
	if err != nil {
		log.Printf("Failed to format generated source code: %s\n%s", err, g.buf.String())
	}
	return src
}
