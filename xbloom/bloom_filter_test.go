package xbloom

import (
	"testing"
)

// TestNew 测试创建新的布隆过滤器
func TestNew(t *testing.T) {
	tests := []struct {
		name               string
		expectedItems      uint64
		falsePositiveRate  float64
		wantSizeNotZero    bool
		wantHashNumNotZero bool
	}{
		{
			name:               "小规模布隆过滤器",
			expectedItems:      100,
			falsePositiveRate:  0.01,
			wantSizeNotZero:    true,
			wantHashNumNotZero: true,
		},
		{
			name:               "中等规模布隆过滤器",
			expectedItems:      10000,
			falsePositiveRate:  0.001,
			wantSizeNotZero:    true,
			wantHashNumNotZero: true,
		},
		{
			name:               "大规模布隆过滤器",
			expectedItems:      1000000,
			falsePositiveRate:  0.0001,
			wantSizeNotZero:    true,
			wantHashNumNotZero: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bf := New(tt.expectedItems, tt.falsePositiveRate)

			if bf == nil {
				t.Fatal("New() 返回 nil")
			}

			if tt.wantSizeNotZero && bf.size == 0 {
				t.Errorf("期望 size > 0, 实际得到 %d", bf.size)
			}

			if tt.wantHashNumNotZero && bf.hashNum == 0 {
				t.Errorf("期望 hashNum > 0, 实际得到 %d", bf.hashNum)
			}

			if bf.bits == nil {
				t.Error("bits 数组不应该为 nil")
			}

			expectedByteSize := (bf.size + 7) / 8
			if uint64(len(bf.bits)) != expectedByteSize {
				t.Errorf("bits 数组长度错误，期望 %d, 实际 %d", expectedByteSize, len(bf.bits))
			}
		})
	}
}

// TestAddAndContains 测试添加元素和检查元素是否存在
func TestAddAndContains(t *testing.T) {
	bf := New(1000, 0.01)

	testData := [][]byte{
		[]byte("hello"),
		[]byte("world"),
		[]byte("布隆过滤器"),
		[]byte("bloom filter"),
		[]byte("test data 123"),
	}

	// 添加元素前，所有元素都不应该存在
	for _, data := range testData {
		if bf.Contains(data) {
			t.Errorf("元素 %s 在添加前就存在", string(data))
		}
	}

	// 添加所有测试元素
	for _, data := range testData {
		bf.Add(data)
	}

	// 添加元素后，所有元素都应该存在
	for _, data := range testData {
		if !bf.Contains(data) {
			t.Errorf("元素 %s 添加后检查不存在", string(data))
		}
	}

	// 测试未添加的元素（可能存在假阳性）
	notAddedData := [][]byte{
		[]byte("not added 1"),
		[]byte("not added 2"),
		[]byte("未添加的数据"),
	}

	falsePositives := 0
	for _, data := range notAddedData {
		if bf.Contains(data) {
			falsePositives++
		}
	}

	// 假阳性率不应该太高（允许一定的假阳性）
	falsePositiveRate := float64(falsePositives) / float64(len(notAddedData))
	if falsePositiveRate > 0.5 { // 允许50%的误差范围
		t.Errorf("假阳性率过高: %f", falsePositiveRate)
	}
}

// TestClear 测试清空布隆过滤器
func TestClear(t *testing.T) {
	bf := New(100, 0.01)

	// 添加一些元素
	testData := [][]byte{
		[]byte("test1"),
		[]byte("test2"),
		[]byte("test3"),
	}

	for _, data := range testData {
		bf.Add(data)
	}

	// 确保元素存在
	for _, data := range testData {
		if !bf.Contains(data) {
			t.Errorf("元素 %s 应该存在", string(data))
		}
	}

	// 清空布隆过滤器
	bf.Clear()

	// 检查所有位是否被清零
	for _, b := range bf.bits {
		if b != 0 {
			t.Error("Clear() 后仍有非零字节")
			break
		}
	}

	// 元素应该不再存在
	for _, data := range testData {
		if bf.Contains(data) {
			t.Errorf("元素 %s 在清空后仍然存在", string(data))
		}
	}
}

// TestGetters 测试获取器方法
func TestGetters(t *testing.T) {
	expectedItems := uint64(1000)
	falsePositiveRate := 0.01
	bf := New(expectedItems, falsePositiveRate)

	// 测试 Size()
	size := bf.Size()
	if size == 0 {
		t.Error("Size() 返回 0")
	}
	if size != bf.size {
		t.Errorf("Size() 返回值不正确，期望 %d, 实际 %d", bf.size, size)
	}

	// 测试 HashFunctions()
	hashNum := bf.HashFunctions()
	if hashNum == 0 {
		t.Error("HashFunctions() 返回 0")
	}
	if hashNum != bf.hashNum {
		t.Errorf("HashFunctions() 返回值不正确，期望 %d, 实际 %d", bf.hashNum, hashNum)
	}
}

// TestEstimatedFalsePositiveRate 测试假阳性率估算
func TestEstimatedFalsePositiveRate(t *testing.T) {
	bf := New(100, 0.01)

	// 空的布隆过滤器假阳性率应该为0
	rate := bf.EstimatedFalsePositiveRate()
	if rate != 0 {
		t.Errorf("空布隆过滤器的假阳性率应该为0，实际为 %f", rate)
	}

	// 添加一些元素
	for i := 0; i < 50; i++ {
		data := []byte{byte(i)}
		bf.Add(data)
	}

	// 添加元素后假阳性率应该大于0
	rate = bf.EstimatedFalsePositiveRate()
	if rate <= 0 {
		t.Errorf("添加元素后假阳性率应该大于0，实际为 %f", rate)
	}

	if rate > 1 {
		t.Errorf("假阳性率不应该大于1，实际为 %f", rate)
	}
}

// TestSerializeDeserialize 测试序列化和反序列化
func TestSerializeDeserialize(t *testing.T) {
	// 创建原始布隆过滤器
	original := New(1000, 0.01)

	// 添加一些测试数据
	testData := [][]byte{
		[]byte("serialize test 1"),
		[]byte("serialize test 2"),
		[]byte("序列化测试"),
		[]byte("数据持久化"),
	}

	for _, data := range testData {
		original.Add(data)
	}

	// 序列化
	serialized, err := original.Serialize()
	if err != nil {
		t.Fatalf("序列化失败: %v", err)
	}

	if len(serialized) == 0 {
		t.Fatal("序列化结果为空")
	}

	// 反序列化
	deserialized, err := Deserialize(serialized)
	if err != nil {
		t.Fatalf("反序列化失败: %v", err)
	}

	if deserialized == nil {
		t.Fatal("反序列化结果为 nil")
	}

	// 检查基本属性是否一致
	if deserialized.size != original.size {
		t.Errorf("反序列化后 size 不一致，期望 %d, 实际 %d", original.size, deserialized.size)
	}

	if deserialized.hashNum != original.hashNum {
		t.Errorf("反序列化后 hashNum 不一致，期望 %d, 实际 %d", original.hashNum, deserialized.hashNum)
	}

	if len(deserialized.bits) != len(original.bits) {
		t.Errorf("反序列化后 bits 长度不一致，期望 %d, 实际 %d", len(original.bits), len(deserialized.bits))
	}

	// 检查位数组内容是否一致
	for i, b := range original.bits {
		if deserialized.bits[i] != b {
			t.Errorf("反序列化后 bits[%d] 不一致，期望 %d, 实际 %d", i, b, deserialized.bits[i])
		}
	}

	// 检查添加的元素在反序列化后是否仍然存在
	for _, data := range testData {
		if !deserialized.Contains(data) {
			t.Errorf("元素 %s 在反序列化后不存在", string(data))
		}
	}

	// 测试反序列化后的功能是否正常
	newData := []byte("新添加的数据")
	deserialized.Add(newData)
	if !deserialized.Contains(newData) {
		t.Error("反序列化后的布隆过滤器无法正常添加新元素")
	}
}

// TestDeserializeErrors 测试反序列化错误情况
func TestDeserializeErrors(t *testing.T) {
	tests := []struct {
		name        string
		data        []byte
		expectError bool
	}{
		{
			name:        "空数据",
			data:        []byte{},
			expectError: true,
		},
		{
			name:        "nil 数据",
			data:        nil,
			expectError: true,
		},
		{
			name:        "无效数据格式",
			data:        []byte{1, 2, 3, 4, 5},
			expectError: true,
		},
		{
			name:        "损坏的 gob 数据",
			data:        []byte("invalid gob data"),
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bf, err := Deserialize(tt.data)

			if tt.expectError {
				if err == nil {
					t.Error("期望出现错误但没有错误")
				}
				if bf != nil {
					t.Error("期望返回 nil 但返回了非 nil 值")
				}
			} else {
				if err != nil {
					t.Errorf("不期望出现错误但发生了错误: %v", err)
				}
				if bf == nil {
					t.Error("期望返回非 nil 值但返回了 nil")
				}
			}
		})
	}
}

// TestSerializeEmpty 测试序列化空的布隆过滤器
func TestSerializeEmpty(t *testing.T) {
	bf := New(100, 0.01)

	// 序列化空的布隆过滤器
	serialized, err := bf.Serialize()
	if err != nil {
		t.Fatalf("序列化空布隆过滤器失败: %v", err)
	}

	// 反序列化
	deserialized, err := Deserialize(serialized)
	if err != nil {
		t.Fatalf("反序列化空布隆过滤器失败: %v", err)
	}

	// 检查属性是否一致
	if deserialized.size != bf.size {
		t.Errorf("空布隆过滤器反序列化后 size 不一致")
	}

	if deserialized.hashNum != bf.hashNum {
		t.Errorf("空布隆过滤器反序列化后 hashNum 不一致")
	}

	// 检查所有位都为0
	for i, b := range deserialized.bits {
		if b != 0 {
			t.Errorf("空布隆过滤器反序列化后 bits[%d] 应该为0，实际为 %d", i, b)
		}
	}
}

// TestEdgeCases 测试边界情况
func TestEdgeCases(t *testing.T) {
	t.Run("极小参数", func(t *testing.T) {
		// 测试极小的预期元素数量
		bf := New(1, 0.5)
		if bf == nil {
			t.Fatal("创建布隆过滤器失败")
		}

		// 基本功能应该正常
		data := []byte("test")
		bf.Add(data)
		if !bf.Contains(data) {
			t.Error("添加的元素应该存在")
		}
	})

	t.Run("极高假阳性率", func(t *testing.T) {
		// 测试接近1.0的假阳性率
		bf := New(100, 0.99)
		if bf == nil {
			t.Fatal("创建布隆过滤器失败")
		}

		// 应该仍然能正常工作
		data := []byte("test")
		bf.Add(data)
		if !bf.Contains(data) {
			t.Error("添加的元素应该存在")
		}
	})

	t.Run("极低假阳性率", func(t *testing.T) {
		// 测试非常小的假阳性率
		bf := New(100, 0.000001)
		if bf == nil {
			t.Fatal("创建布隆过滤器失败")
		}

		// 应该创建较大的位数组
		if bf.size == 0 {
			t.Error("位数组大小应该大于0")
		}

		data := []byte("test")
		bf.Add(data)
		if !bf.Contains(data) {
			t.Error("添加的元素应该存在")
		}
	})
}

// TestLargeData 测试大数据处理
func TestLargeData(t *testing.T) {
	t.Run("大量元素", func(t *testing.T) {
		bf := New(10000, 0.01)

		// 添加大量元素
		numElements := 5000
		addedElements := make([][]byte, numElements)

		for i := 0; i < numElements; i++ {
			data := []byte("element_" + string(rune(i)))
			addedElements[i] = data
			bf.Add(data)
		}

		// 检查所有添加的元素都存在
		missingCount := 0
		for _, data := range addedElements {
			if !bf.Contains(data) {
				missingCount++
			}
		}

		if missingCount > 0 {
			t.Errorf("有 %d 个已添加的元素检查时不存在", missingCount)
		}
	})

	t.Run("大数据块", func(t *testing.T) {
		bf := New(100, 0.01)

		// 测试大的数据块
		largeData := make([]byte, 10000)
		for i := range largeData {
			largeData[i] = byte(i % 256)
		}

		bf.Add(largeData)
		if !bf.Contains(largeData) {
			t.Error("大数据块添加后应该存在")
		}

		// 测试略有不同的大数据块
		largeData2 := make([]byte, 10000)
		copy(largeData2, largeData)
		largeData2[5000] = largeData2[5000] + 1 // 修改一个字节

		// 这个应该不存在（除非出现假阳性）
		// 但我们不强制要求，因为可能出现假阳性
	})
}

// TestConcurrency 测试并发安全性（注意：当前实现不是并发安全的）
func TestConcurrency(t *testing.T) {
	t.Skip("当前实现不支持并发，跳过并发测试")

	// 如果将来实现并发安全，可以启用这个测试
	/*
		bf := New(1000, 0.01)

		var wg sync.WaitGroup
		numGoroutines := 10
		elementsPerGoroutine := 100

		// 并发添加元素
		for g := 0; g < numGoroutines; g++ {
			wg.Add(1)
			go func(goroutineID int) {
				defer wg.Done()
				for i := 0; i < elementsPerGoroutine; i++ {
					data := []byte(fmt.Sprintf("goroutine_%d_element_%d", goroutineID, i))
					bf.Add(data)
				}
			}(g)
		}

		wg.Wait()

		// 验证所有元素都被正确添加
		for g := 0; g < numGoroutines; g++ {
			for i := 0; i < elementsPerGoroutine; i++ {
				data := []byte(fmt.Sprintf("goroutine_%d_element_%d", g, i))
				if !bf.Contains(data) {
					t.Errorf("并发添加的元素 %s 不存在", string(data))
				}
			}
		}
	*/
}

// TestHashFunctions 测试哈希函数
func TestHashFunctions(t *testing.T) {
	t.Run("哈希函数一致性", func(t *testing.T) {
		data := []byte("consistent test data")

		// 同样的数据应该产生同样的哈希值
		h1_1 := hash1(data)
		h1_2 := hash1(data)
		if h1_1 != h1_2 {
			t.Error("hash1 对相同数据产生不同结果")
		}

		h2_1 := hash2(data)
		h2_2 := hash2(data)
		if h2_1 != h2_2 {
			t.Error("hash2 对相同数据产生不同结果")
		}
	})

	t.Run("不同哈希函数产生不同结果", func(t *testing.T) {
		data := []byte("test different hash functions")

		h1 := hash1(data)
		h2 := hash2(data)

		// 两个哈希函数对同样数据应该产生不同结果（大概率）
		if h1 == h2 {
			t.Log("警告：两个哈希函数产生了相同结果，这可能会影响布隆过滤器性能")
		}
	})

	t.Run("空数据哈希", func(t *testing.T) {
		emptyData := []byte{}

		h1 := hash1(emptyData)
		h2 := hash2(emptyData)

		// 空数据也应该产生有效的哈希值
		if h1 == 0 && h2 == 0 {
			t.Error("两个哈希函数对空数据都返回0，这可能不是期望的行为")
		}
	})
}

// BenchmarkAdd 性能测试：添加元素
func BenchmarkAdd(b *testing.B) {
	bf := New(uint64(b.N), 0.01)
	data := []byte("benchmark test data")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bf.Add(data)
	}
}

// BenchmarkContains 性能测试：检查元素
func BenchmarkContains(b *testing.B) {
	bf := New(1000, 0.01)
	data := []byte("benchmark test data")
	bf.Add(data)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bf.Contains(data)
	}
}

// BenchmarkSerialize 性能测试：序列化
func BenchmarkSerialize(b *testing.B) {
	bf := New(10000, 0.01)
	// 添加一些数据
	for i := 0; i < 1000; i++ {
		data := []byte("benchmark_" + string(rune(i)))
		bf.Add(data)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := bf.Serialize()
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkDeserialize 性能测试：反序列化
func BenchmarkDeserialize(b *testing.B) {
	bf := New(10000, 0.01)
	// 添加一些数据
	for i := 0; i < 1000; i++ {
		data := []byte("benchmark_" + string(rune(i)))
		bf.Add(data)
	}

	serialized, err := bf.Serialize()
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := Deserialize(serialized)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func Test_optimalSize(t *testing.T) {
	type args struct {
		n uint64
		p float64
	}
	tests := []struct {
		name string
		args args
		want uint64
	}{
		{
			name: "1",
			args: args{
				n: 10000,
				p: 1e-8, // 1亿分之一的假阳
			},
			want: 383403,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := optimalSize(tt.args.n, tt.args.p); got != tt.want {
				t.Errorf("optimalSize() = %v, want %v", got, tt.want)
			}
		})
	}
}
