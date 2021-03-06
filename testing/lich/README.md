## testing/lich 运行环境构建
基于 docker-compose 实现跨平台跨语言环境的容器依赖管理方案，以解决运行 unit test 场景下的 (mysql, redis, mc)容器依赖问题。

# 背景

单元测试即对最小可测试单元进行检查和验证，它可以很好的让你的代码在上测试环境之前自己就能前置的发现问题，解决问题。当然每个语言都有原生支持的 UT 框架，不过在 Kratos 里面我们需要有一些配套设施以及周边工具来辅助我们构筑整个 UT 生态。

# 工具链

- testgen UT代码自动生成器（README: tool/testgen/README.md）
- testcli UT运行环境构建工具（README: tool/testcli/README.md）

# 测试框架选型

golang 的单元测试，既可以用官方自带的 testing 包，也有开源的如 testify、goconvey 业内知名，使用非常多也很好用的框架。

根据一番调研和内部使用经验，我们确定：

> - testing 作为基础库测试框架（非常精简不过够用）
> - goconvey 作为业务程序的单元测试框架（因为涉及比较多的业务场景和流程控制判断，比如更丰富的res值判断、上下文嵌套支持、还有webUI等）

# 单元测试标准

1. 覆盖率，当前标准：60%（所有包均需达到） 尽量达到70%以上。当然覆盖率并不能完全说明单元测试的质量，开发者需要考虑关键的条件判断和预期的结果。复杂的代码是需要好好设计测试用例的。
2. 通过率，当前标准：100%（所有用例中的断言必须通过）

# 书写建议

1. 结果验证

   > - 校验err是否为nil. err是go函数的标配了，也是最基础的判断，如果err不为nil，基本上函数返回值或者处理肯定是有问题了。
   > - 检验res值是否正确。res值的校验是非常重要的，也是很容易忽略的地方。比如返回结构体对象，要对结构体的成员进行判断，而有可能里面是0值。goconvey对res值的判断支持是非常友好的。

2. 逻辑验证

   > 业务代码经常是流程比较复杂的，而函数的执行结果也是有上下文的，比如有不同条件分支。goconvey就非常优雅的支持了这种情况，可以嵌套执行。单元测试要结合业务代码逻辑，才能尽量的减少线上bug。

3. 如何mock 主要分以下3块：

   > - 基础组件，如mc、redis、mysql等，由 testcli(testing/lich) 起基础镜像支持（需要提供建表、INSERT语句）与本地开发环境一致，也保证了结果的一致性。
   > - rpc server，如 xxxx-service 需要定义 interface 供业务依赖方使用。所有rpc server 都必须要提供一个interface+mock代码(gomock)。
   > - http server则直接写mock代码gock。


