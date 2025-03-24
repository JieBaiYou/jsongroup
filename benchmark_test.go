package jsongroup

import (
	"encoding/json"
	"testing"
)

// 各种规模的测试数据
type BenchUser struct {
	ID       int    `json:"id" groups:"public,admin"`
	Name     string `json:"name,omitempty" groups:"public,admin"`
	Email    string `json:"email" groups:"admin"`
	Password string `json:"password" groups:"internal"`
	Age      int    `json:"age" groups:"public,admin"`
	Address  string `json:"address,omitempty" groups:"public"`
	Phone    string `json:"phone" groups:"admin"`
	Role     string `json:"role" groups:"admin"`
	Username string `json:"username" groups:"public"`
	Avatar   string `json:"avatar,omitempty" groups:"public"`
	Bio      string `json:"bio,omitempty" groups:"public"`
}

// 嵌套结构
type BenchProfile struct {
	Skills  []string          `json:"skills" groups:"public"`
	Socials map[string]string `json:"socials" groups:"public"`
	Stats   struct {
		Posts    int `json:"posts" groups:"public"`
		Likes    int `json:"likes" groups:"public"`
		Comments int `json:"comments" groups:"admin"`
	} `json:"stats" groups:"public,admin"`
	Settings *struct {
		Theme      string `json:"theme" groups:"public"`
		Visibility string `json:"visibility" groups:"admin"`
	} `json:"settings,omitempty" groups:"public,admin"`
}

type BenchComplexUser struct {
	BenchUser
	Profile BenchProfile `json:"profile" groups:"public,admin"`
	Posts   []struct {
		ID      int    `json:"id" groups:"public"`
		Title   string `json:"title" groups:"public"`
		Content string `json:"content" groups:"public"`
	} `json:"posts" groups:"public"`
	Followers []*BenchUser `json:"followers,omitempty" groups:"admin"`
}

// 基准测试数据生成
func createTestUser() BenchUser {
	return BenchUser{
		ID:       1,
		Name:     "Test User",
		Email:    "test@example.com",
		Password: "password123",
		Age:      25,
		Address:  "123 Test Street",
		Phone:    "123-456-7890",
		Role:     "user",
		Username: "testuser",
		Avatar:   "https://example.com/avatar.jpg",
		Bio:      "This is a test user bio with some text for testing purposes.",
	}
}

func createComplexUser() BenchComplexUser {
	user := createTestUser()

	// 创建测试数据
	complex := BenchComplexUser{
		BenchUser: user,
		Profile: BenchProfile{
			Skills: []string{"Go", "JavaScript", "Python", "SQL"},
			Socials: map[string]string{
				"github":   "https://github.com/testuser",
				"twitter":  "https://twitter.com/testuser",
				"linkedin": "https://linkedin.com/in/testuser",
			},
		},
		Posts: make([]struct {
			ID      int    `json:"id" groups:"public"`
			Title   string `json:"title" groups:"public"`
			Content string `json:"content" groups:"public"`
		}, 5),
	}

	// 初始化结构体里的结构体
	complex.Profile.Stats.Posts = 42
	complex.Profile.Stats.Likes = 123
	complex.Profile.Stats.Comments = 15

	// 初始化指针字段
	settings := struct {
		Theme      string `json:"theme" groups:"public"`
		Visibility string `json:"visibility" groups:"admin"`
	}{
		Theme:      "dark",
		Visibility: "public",
	}
	complex.Profile.Settings = &settings

	// 添加测试文章
	for i := 0; i < 5; i++ {
		complex.Posts[i] = struct {
			ID      int    `json:"id" groups:"public"`
			Title   string `json:"title" groups:"public"`
			Content string `json:"content" groups:"public"`
		}{
			ID:      i + 1,
			Title:   "Test Post " + string(rune('A'+i)),
			Content: "This is the content of test post " + string(rune('A'+i)),
		}
	}

	return complex
}

// 基准测试：基本结构体，公开分组
func BenchmarkMarshalSimpleUserPublic(b *testing.B) {
	user := createTestUser()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := MarshalByGroups(user, "public")
		if err != nil {
			b.Fatal(err)
		}
	}
}

// 基准测试：基本结构体，标准json
func BenchmarkStandardJSONMarshalSimpleUser(b *testing.B) {
	user := createTestUser()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := json.Marshal(user)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// 基准测试：复杂嵌套结构体，公开分组
func BenchmarkMarshalComplexUserPublic(b *testing.B) {
	complex := createComplexUser()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := MarshalByGroups(complex, "public")
		if err != nil {
			b.Fatal(err)
		}
	}
}

// 基准测试：复杂嵌套结构体，标准json
func BenchmarkStandardJSONMarshalComplexUser(b *testing.B) {
	complex := createComplexUser()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := json.Marshal(complex)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// 基准测试：复杂结构体，多组
func BenchmarkMarshalComplexUserMultipleGroups(b *testing.B) {
	complex := createComplexUser()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := MarshalByGroups(complex, "public", "admin")
		if err != nil {
			b.Fatal(err)
		}
	}
}

// 基准测试：启用缓存
func BenchmarkMarshalWithCache(b *testing.B) {
	user := createTestUser()

	// 预热缓存
	_, _ = MarshalByGroups(user, "public")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := MarshalByGroups(user, "public")
		if err != nil {
			b.Fatal(err)
		}
	}
}

// 基准测试：禁用缓存
func BenchmarkMarshalWithoutCache(b *testing.B) {
	user := createTestUser()
	originalSize := DefaultMaxCacheSize

	// 禁用缓存
	SetMaxCacheSize(0)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := MarshalByGroups(user, "public")
		if err != nil {
			b.Fatal(err)
		}
	}

	// 恢复缓存设置
	b.StopTimer()
	SetMaxCacheSize(originalSize)
}

// 基准测试：复杂结构体，Map序列化
func BenchmarkMarshalToMapComplexUser(b *testing.B) {
	complex := createComplexUser()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := MarshalToMap(complex, "public")
		if err != nil {
			b.Fatal(err)
		}
	}
}
