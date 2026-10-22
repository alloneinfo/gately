package guard

import "github.com/wangwanjie/gately/internal/events"

// Presets 是常见服务的规则模板，界面按 Preset 字段翻译名称与说明。
// 过滤规则参考 fail2ban 的同名过滤器，按 Go 正则（RE2）改写并精简。
var Presets = []Jail{
	{
		Preset: "sshd", Name: "SSH", Source: SourceFile, Target: "/var/log/auth.log",
		Filter:   sshdFilter,
		MaxRetry: 5, FindTime: 600,
	},
	{
		Preset: "sshd-journal", Name: "SSH (journald)", Source: SourceJournal, Target: "ssh.service\nsshd.service",
		Filter:   sshdFilter,
		MaxRetry: 5, FindTime: 600,
	},
	{
		Preset: "dropbear", Name: "Dropbear", Source: SourceCommand, Target: "logread -f",
		Filter: `Bad password attempt for '.*' from <HOST>:\d+\s*$
Login attempt for nonexistent user from <HOST>:\d+\s*$`,
		MaxRetry: 5, FindTime: 600,
	},
	{
		Preset: "nginx-http-auth", Name: "Nginx Basic Auth", Source: SourceFile, Target: "/var/log/nginx/error.log",
		Filter:   `user ".*":? (?:password mismatch|was not found in "[^"]*"), client: <HOST>, server: \S*, request: "\S+ \S+ HTTP/[\d.]+"(?:, host: "\S+")?(?:, referrer: "\S+")?\s*$`,
		MaxRetry: 5, FindTime: 600,
	},
	{
		Preset: "nginx-botsearch", Name: "Nginx Bot Search", Source: SourceFile, Target: "/var/log/nginx/access.log",
		Filter:   `^<HOST> \S+ \S+ \[[^\]]+\] "(?:GET|POST|HEAD) [^"]*(?:wp-login\.php|xmlrpc\.php|/wp-admin|phpmyadmin|/pma/|/\.env|/\.git/|/cgi-bin/|/boaform/|/HNAP1|/vendor/phpunit|/actuator/)[^"]*" (?:400|401|403|404|405) `,
		MaxRetry: 3, FindTime: 600, BanTime: 86400,
	},
	{
		Preset: "vaultwarden", Name: "Vaultwarden", Source: SourceDocker, Target: "vaultwarden",
		Filter: `Username or password is incorrect\. Try again\. IP: <HOST>\. Username:
Invalid TOTP code! Server time: .* IP: <HOST>\s*$`,
		MaxRetry: 5, FindTime: 600,
	},
	{
		// 主机名部分来自反向解析，可被攻击者控制，只取括号中的地址
		Preset: "homeassistant", Name: "Home Assistant", Source: SourceDocker, Target: "homeassistant",
		Filter:   `Login attempt or request with invalid authentication from \S+ \(<HOST>\)`,
		MaxRetry: 5, FindTime: 600,
	},
	{
		Preset: "proxmox", Name: "Proxmox VE", Source: SourceJournal, Target: "pvedaemon.service",
		Filter:   `authentication (?:failure|error); rhost=(?:::ffff:)?<HOST> user=`,
		MaxRetry: 5, FindTime: 600,
	},
	{
		Preset: "gately-admin", Name: "Gately 管理后台", Source: SourceEvent, Target: events.AuthFailed,
		MaxRetry: 5, FindTime: 600,
	},
	{
		Preset: "gately-waf", Name: "WAF 拦截", Source: SourceEvent, Target: events.WAFBlocked,
		MaxRetry: 5, FindTime: 300, BanTime: 86400,
	},
}

// sshdFilter 是 OpenSSH 的失败记录。用户名可被攻击者任意填写（例如包含 “from 8.8.8.8”），
// 因此每条都锚定到行尾，取最后出现的来源地址，避免借日志注入封禁任意地址。
const sshdFilter = `Failed (?:password|publickey) for (?:invalid user )?.* from <HOST> port \d+(?: ssh\d?)?\s*$
Invalid user .* from <HOST> port \d+\s*$
(?:Connection closed|Disconnected) by (?:invalid |authenticating )?user .* <HOST> port \d+ \[preauth\]\s*$
maximum authentication attempts exceeded for .* from <HOST> port \d+(?: ssh\d?)?(?: \[preauth\])?\s*$
Did not receive identification string from <HOST>(?: port \d+)?\s*$
banner exchange: Connection from <HOST> port \d+: invalid format\s*$`
