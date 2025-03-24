package jsongroup

import (
	"encoding/json"
	"reflect"
	"testing"
)

type Address struct {
	Street string `json:"street" groups:"admin,public"`
	City   string `json:"city" groups:"admin,public"`
	ZIP    string `json:"zip" groups:"admin"`
}

type User struct {
	ID       int            `json:"id" groups:"public,admin"`
	Name     string         `json:"name,omitempty" groups:"public,admin"`
	Email    string         `json:"email" groups:"admin,internal"`
	Password string         `json:"password" groups:"internal"`
	Address  *Address       `json:"address" groups:"public,admin"`
	Tags     []string       `json:"tags" groups:"public"`
	Settings map[string]any `json:"settings" groups:"admin"`
}

type BaseInfo struct {
	CreatedAt string `json:"created_at" groups:"admin"`
	UpdatedAt string `json:"updated_at" groups:"admin"`
}

type Profile struct {
	BaseInfo        // 嵌入BaseInfo字段
	Age      int    `json:"age" groups:"public,admin"`
	Bio      string `json:"bio" groups:"public"`
	Private  bool   `json:"private" groups:"admin"`
}

type ComplexUser struct {
	User                                           // 嵌入User字段
	Profile `json:"profile" groups:"public,admin"` // 命名嵌入字段
}

func TestMarshalByGroups(t *testing.T) {
	// 测试用例
	tests := []struct {
		name    string
		value   any
		groups  []string
		want    string
		wantErr bool
	}{
		{
			name: "基本用户对象-公开分组",
			value: User{
				ID:       1,
				Name:     "张三",
				Email:    "zhangsan@example.com",
				Password: "secret",
				Address: &Address{
					Street: "中关村大街1号",
					City:   "北京",
					ZIP:    "100080",
				},
				Tags: []string{"VIP", "新用户"},
				Settings: map[string]any{
					"theme":         "dark",
					"notifications": true,
				},
			},
			groups: []string{"public"},
			want:   `{"address":{"city":"北京","street":"中关村大街1号"},"id":1,"name":"张三","tags":["VIP","新用户"]}`,
		},
		{
			name: "基本用户对象-管理员分组",
			value: User{
				ID:       1,
				Name:     "张三",
				Email:    "zhangsan@example.com",
				Password: "secret",
				Address: &Address{
					Street: "中关村大街1号",
					City:   "北京",
					ZIP:    "100080",
				},
				Tags: []string{"VIP", "新用户"},
				Settings: map[string]any{
					"theme":         "dark",
					"notifications": true,
				},
			},
			groups: []string{"admin"},
			want:   `{"address":{"city":"北京","street":"中关村大街1号","zip":"100080"},"email":"zhangsan@example.com","id":1,"name":"张三","settings":{"notifications":true,"theme":"dark"}}`,
		},
		{
			name: "基本用户对象-内部分组",
			value: User{
				ID:       1,
				Name:     "张三",
				Email:    "zhangsan@example.com",
				Password: "secret",
				Address: &Address{
					Street: "中关村大街1号",
					City:   "北京",
					ZIP:    "100080",
				},
			},
			groups: []string{"internal"},
			want:   `{"email":"zhangsan@example.com","password":"secret"}`,
		},
		{
			name: "复杂用户对象-公开分组",
			value: ComplexUser{
				User: User{
					ID:    1,
					Name:  "张三",
					Email: "zhangsan@example.com",
					Address: &Address{
						Street: "中关村大街1号",
						City:   "北京",
						ZIP:    "100080",
					},
					Tags: []string{"VIP"},
				},
				Profile: Profile{
					BaseInfo: BaseInfo{
						CreatedAt: "2023-01-01",
						UpdatedAt: "2023-05-01",
					},
					Age:     30,
					Bio:     "软件工程师",
					Private: true,
				},
			},
			groups: []string{"public"},
			want:   `{"address":{"city":"北京","street":"中关村大街1号"},"age":30,"bio":"软件工程师","id":1,"name":"张三","tags":["VIP"]}`,
		},
		{
			name: "带顶层键包装",
			value: User{
				ID:   1,
				Name: "张三",
			},
			groups: []string{"public"},
			want:   `{"id":1,"name":"张三","tags":[]}`,
		},
		{
			name: "nil指针字段",
			value: User{
				ID:      1,
				Name:    "张三",
				Address: nil,
			},
			groups: []string{"public"},
			want:   `{"id":1,"name":"张三","tags":[]}`,
		},
		{
			name:    "nil值",
			value:   nil,
			groups:  []string{"public"},
			want:    `null`,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := MarshalByGroups(tt.value, tt.groups...)
			if (err != nil) != tt.wantErr {
				t.Errorf("MarshalByGroups() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			var gotMap, wantMap any
			if err := json.Unmarshal(got, &gotMap); err != nil {
				t.Errorf("无法解析生成的JSON: %v", err)
			}
			if err := json.Unmarshal([]byte(tt.want), &wantMap); err != nil {
				t.Errorf("无法解析期望的JSON: %v", err)
			}

			if !reflect.DeepEqual(gotMap, wantMap) {
				t.Errorf("MarshalByGroups() = %s, want %s", string(got), tt.want)
			}
		})
	}
}

func TestMarshalByGroupsWithOptions(t *testing.T) {
	// 移除未使用的变量
	user := User{
		ID:    1,
		Name:  "张三",
		Email: "zhangsan@example.com",
	}

	// 测试顶层包装
	t.Run("TopLevelKey", func(t *testing.T) {
		// 启用忽略nil指针
		got, err := MarshalByGroupsWithOptions(user, DefaultOptions().WithTopLevelKey("user"), "public")
		if err != nil {
			t.Errorf("MarshalByGroupsWithOptions() error = %v", err)
			return
		}

		want := `{"user":{"id":1,"name":"张三","tags":[]}}`
		var gotMap, wantMap any
		_ = json.Unmarshal(got, &gotMap)
		_ = json.Unmarshal([]byte(want), &wantMap)

		if !reflect.DeepEqual(gotMap, wantMap) {
			t.Errorf("MarshalByGroupsWithOptions() = %s, want %s", string(got), want)
		}
	})

	// 测试AND逻辑（必须同时满足所有分组）
	t.Run("GroupModeAnd", func(t *testing.T) {
		opts := DefaultOptions().WithGroupMode(GroupModeAnd)

		user := User{
			ID:      1,
			Name:    "张三",
			Email:   "zhangsan@example.com",
			Address: nil,
		}

		got, err := MarshalByGroupsWithOptions(user, opts, "public", "admin")
		if err != nil {
			t.Errorf("MarshalByGroupsWithOptions() error = %v", err)
			return
		}

		// 只有ID和Name字段同时属于public和admin分组
		want := `{"id":1,"name":"张三"}`
		var gotMap, wantMap any
		_ = json.Unmarshal(got, &gotMap)
		_ = json.Unmarshal([]byte(want), &wantMap)

		if !reflect.DeepEqual(gotMap, wantMap) {
			t.Errorf("MarshalByGroupsWithOptions() = %s, want %s", string(got), want)
		}
	})

	// 测试NullIfEmpty选项
	t.Run("NullIfEmpty", func(t *testing.T) {
		user := User{
			ID:      1,
			Name:    "张三",
			Address: nil,
			Tags:    nil, // 显式设置为nil
		}

		// 使用NullIfEmpty选项（同时会禁用IgnoreNilPointers）
		opts := DefaultOptions().WithNullIfEmpty(true)
		got, err := MarshalByGroupsWithOptions(user, opts, "public")
		if err != nil {
			t.Errorf("MarshalByGroupsWithOptions() error = %v", err)
			return
		}

		// 输出预期结果以帮助调试
		t.Logf("NullIfEmpty 实际输出: %s", string(got))

		// Address字段应该为null而不是被省略，Tags字段为nil所以也应该为null
		want := `{"address":null,"id":1,"name":"张三","tags":null}`
		var gotMap, wantMap any
		_ = json.Unmarshal(got, &gotMap)
		_ = json.Unmarshal([]byte(want), &wantMap)

		if !reflect.DeepEqual(gotMap, wantMap) {
			t.Errorf("MarshalByGroupsWithOptions() = %s, want %s", string(got), want)
		}
	})

	// 测试忽略nil指针
	t.Run("IgnoreNilPointers", func(t *testing.T) {
		user := User{
			ID:      1,
			Name:    "张三",
			Address: nil,
			Tags:    []string{},
		}

		// 显式启用IgnoreNilPointers选项
		opts := DefaultOptions().WithIgnoreNilPointers(true)
		got, err := MarshalByGroupsWithOptions(user, opts, "public")
		if err != nil {
			t.Errorf("MarshalByGroupsWithOptions() error = %v", err)
			return
		}

		// Address字段应该被完全忽略
		want := `{"id":1,"name":"张三","tags":[]}`
		var gotMap, wantMap any
		_ = json.Unmarshal(got, &gotMap)
		_ = json.Unmarshal([]byte(want), &wantMap)

		if !reflect.DeepEqual(gotMap, wantMap) {
			t.Errorf("MarshalByGroupsWithOptions() = %s, want %s", string(got), want)
		}
	})
}

func TestMarshalToMap(t *testing.T) {
	user := User{
		ID:   1,
		Name: "张三",
		Address: &Address{
			Street: "中关村大街1号",
			City:   "北京",
		},
	}

	got, err := MarshalToMap(user, "public")
	if err != nil {
		t.Errorf("MarshalToMap() error = %v", err)
		return
	}

	// 检查结果是否是map
	if _, ok := got["id"]; !ok {
		t.Errorf("MarshalToMap() 结果中找不到字段 'id'")
	}
	if _, ok := got["address"]; !ok {
		t.Errorf("MarshalToMap() 结果中找不到字段 'address'")
	}

	// 检查嵌套结构
	if address, ok := got["address"].(map[string]any); ok {
		if _, ok := address["street"]; !ok {
			t.Errorf("MarshalToMap() 嵌套结构中找不到字段 'street'")
		}
	} else {
		t.Errorf("MarshalToMap() 字段 'address' 不是一个map")
	}
}
