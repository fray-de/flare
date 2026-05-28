package cmd_test

import (
	"os"
	"testing"

	env "github.com/caarlos0/env/v6"
	"github.com/soulteary/flare/cmd"
	"github.com/soulteary/flare/config/define"
	"github.com/soulteary/flare/config/model"
	"github.com/stretchr/testify/assert"
)

func TestParseEnvVars(t *testing.T) {
	os.Setenv("FLARE_PORT", "5000")
	defer os.Unsetenv("FLARE_PORT")

	os.Setenv("FLARE_GUIDE", "false")
	defer os.Unsetenv("FLARE_GUIDE")

	os.Setenv("FLARE_OFFLINE", "true")
	defer os.Unsetenv("FLARE_OFFLINE")

	os.Setenv("FLARE_USER", "test")
	defer os.Unsetenv("FLARE_USER")

	os.Setenv("FLARE_VISIBILITY", "private")
	defer os.Unsetenv("FLARE_VISIBILITY")

	flags := cmd.ParseEnvVars()

	assert.Equal(t, flags.Port, 5000)
	assert.Equal(t, flags.EnableGuide, false)
	assert.Equal(t, flags.EnableOfflineMode, true)
	assert.Equal(t, flags.Visibility, "private")
	assert.Equal(t, flags.User, "test")

	// test error parse
	os.Setenv("FLARE_OFFLINE", ")))))))@#$%^&*()")
	defer os.Unsetenv("FLARE_OFFLINE")
	flags = cmd.ParseEnvVars()
	defaultEnvs := define.DefaultEnvVars
	assert.Equal(t, flags.EnableOfflineMode, defaultEnvs.EnableOfflineMode)
}

// TestParseEnvVars_CookieDefaultsAndOverride 回归: CookieSecret/CookieName 必须被合并,
// 否则 CookieSecret 为空会导致登录时"保存登陆状态失败"。
func TestParseEnvVars_CookieDefaultsAndOverride(t *testing.T) {
	// 默认: 未设置环境变量时应回退到内置默认值(非空)
	os.Unsetenv("FLARE_COOKIE_SECRET")
	os.Unsetenv("FLARE_COOKIE_NAME")
	flags := cmd.ParseEnvVars()
	assert.Equal(t, define.DEFAULT_COOKIE_SECRET, flags.CookieSecret, "CookieSecret 默认应为内置默认值且非空")
	assert.NotEmpty(t, flags.CookieSecret, "CookieSecret 不能为空, 否则 cookie 无法签名")
	assert.Equal(t, define.DEFAULT_COOKIE_NAME, flags.CookieName)

	// 覆盖: 设置 FLARE_COOKIE_SECRET 时应被采用(此前会被静默忽略)
	os.Setenv("FLARE_COOKIE_SECRET", "my-strong-secret")
	defer os.Unsetenv("FLARE_COOKIE_SECRET")
	flags = cmd.ParseEnvVars()
	assert.Equal(t, "my-strong-secret", flags.CookieSecret, "FLARE_COOKIE_SECRET 应生效")
}

func TestInitAccountFromEnvVars_normal(t *testing.T) {
	defaultEnvs := define.DefaultEnvVars

	err := env.Parse(&defaultEnvs)
	assert.Nil(t, err, "TestInitAccountFromEnvVars Faild")
	var target model.Flags

	// 3. update username and password
	cmd.InitAccountFromEnvVars(
		"custom",
		defaultEnvs.Pass,
		&target.User,
		&target.Pass,
		define.DEFAULT_USER_NAME,
		&target.UserIsGenerated,
		&target.PassIsGenerated,
		&target.DisableLoginMode,
	)
	assert.Equal(t, target.User, "custom")
	assert.Equal(t, target.UserIsGenerated, false)
	assert.Equal(t, target.PassIsGenerated, true)
	assert.Equal(t, len(target.Pass), 8)
}

func TestInitAccountFromEnvVars_EmptyUser(t *testing.T) {
	defaultEnvs := define.DefaultEnvVars

	err := env.Parse(&defaultEnvs)
	assert.Nil(t, err, "TestInitAccountFromEnvVars Faild")
	var target model.Flags

	// 4. test empty username and password
	cmd.InitAccountFromEnvVars(
		"",
		defaultEnvs.Pass,
		&target.User,
		&target.Pass,
		define.DEFAULT_USER_NAME,
		&target.UserIsGenerated,
		&target.PassIsGenerated,
		&target.DisableLoginMode,
	)
	assert.Equal(t, target.User, define.DEFAULT_USER_NAME)
	assert.Equal(t, target.UserIsGenerated, true)
	assert.Equal(t, target.PassIsGenerated, true)
	assert.Equal(t, len(target.Pass), 8)
}

func TestInitAccountFromEnvVars_EmptyPass(t *testing.T) {
	defaultEnvs := define.DefaultEnvVars

	err := env.Parse(&defaultEnvs)
	assert.Nil(t, err, "TestInitAccountFromEnvVars Faild")

	var target model.Flags

	// 4. test empty password
	cmd.InitAccountFromEnvVars(
		"custom",
		"",
		&target.User,
		&target.Pass,
		define.DEFAULT_USER_NAME,
		&target.UserIsGenerated,
		&target.PassIsGenerated,
		&target.DisableLoginMode,
	)
	assert.Equal(t, target.User, "custom")
	assert.Equal(t, len(target.Pass), 8)
	assert.Equal(t, target.PassIsGenerated, true)
}

func TestInitAccountFromEnvVars_Pass(t *testing.T) {
	defaultEnvs := define.DefaultEnvVars

	err := env.Parse(&defaultEnvs)
	assert.Nil(t, err, "TestInitAccountFromEnvVars Faild")
	var target model.Flags

	// 4. test empty password
	cmd.InitAccountFromEnvVars(
		"custom",
		"custom",
		&target.User,
		&target.Pass,
		define.DEFAULT_USER_NAME,
		&target.UserIsGenerated,
		&target.PassIsGenerated,
		&target.DisableLoginMode,
	)
	assert.Equal(t, target.User, "custom")
	assert.Equal(t, target.Pass, "custom")
	assert.Equal(t, target.PassIsGenerated, false)
	assert.Equal(t, target.UserIsGenerated, false)
}
