package nginx

import (
	"bytes"
	"strings"
	"testing"
	"text/template"

	"github.com/geodro/lerd/internal/config"
)

func renderVhostForTest(t *testing.T, name string, data VhostData) string {
	t.Helper()
	raw, err := GetTemplate(name)
	if err != nil {
		t.Fatalf("GetTemplate(%s): %v", name, err)
	}
	tmpl, err := template.New(name).Parse(string(raw))
	if err != nil {
		t.Fatalf("parse %s: %v", name, err)
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		t.Fatalf("execute %s: %v", name, err)
	}
	return buf.String()
}

func TestIndentBlock(t *testing.T) {
	got := indentBlock("a {\n  b;\n\n}", "    ")
	want := "    a {\n      b;\n\n    }"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestExpandNginxSnippet(t *testing.T) {
	got, err := expandNginxSnippet(
		"root {{root}};\nalias {{public}};\nfastcgi_pass {{fpm}}:9000;",
		"/home/u/shop", "pub", "lerd-php84-fpm",
	)
	if err != nil {
		t.Fatalf("expand: %v", err)
	}
	for _, want := range []string{
		`set $lerd_root "/home/u/shop";`,
		`set $lerd_public "/home/u/shop/pub";`,
		"root ${lerd_root};",
		"alias ${lerd_public};",
		"fastcgi_pass lerd-php84-fpm:9000;",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in:\n%s", want, got)
		}
	}
	if strings.Contains(got, "{{") {
		t.Errorf("unexpanded placeholder in:\n%s", got)
	}
}

// `{{roots}}` has balanced braces, so ValidateNginxSnippet accepts it. If it
// reached nginx verbatim it would break the config for every site on the box.
func TestExpandNginxSnippetRejectsUnknownPlaceholder(t *testing.T) {
	snippet := "root {{roots}};"
	if err := config.ValidateNginxSnippet(snippet); err != nil {
		t.Fatalf("typo has balanced braces, should validate: %v", err)
	}
	if _, err := expandNginxSnippet(snippet, "/p", "pub", "fpm"); err == nil {
		t.Error("unknown placeholder was accepted")
	}
}

// A "." public_dir means the project root is the document root.
func TestExpandNginxSnippetDotPublicDir(t *testing.T) {
	got, err := expandNginxSnippet("root {{public}};", "/home/u/wp", ".", "fpm")
	if err != nil {
		t.Fatalf("expand: %v", err)
	}
	if want := "set $lerd_public \"/home/u/wp\";"; !strings.Contains(got, want) {
		t.Fatalf("got %q, want %q in it", got, want)
	}
}

// Brace balance cannot contain an interpolated value: `}` plus `server {` still
// balances. So hostile values must be rejected at substitution time instead.
func TestExpandNginxSnippetRejectsHostileValues(t *testing.T) {
	snippet := "location /a/ {\n    root {{root}};\n}"
	if err := config.ValidateNginxSnippet(snippet); err != nil {
		t.Fatalf("snippet should be balanced: %v", err)
	}

	escape := "/home/u/x}\nserver { listen 81;"
	if _, err := expandNginxSnippet(snippet, escape, "pub", "fpm"); err == nil {
		t.Error("a path that closes the block and opens a server was accepted")
	}
	// It would have passed a post-expansion brace check, which is the whole point.
	balanced := strings.NewReplacer("{{root}}", escape).Replace(snippet)
	if err := config.ValidateNginxSnippet(balanced); err != nil {
		t.Logf("post-expansion check happened to catch it: %v", err)
	}

	for _, bad := range []string{"/p;deny all", "/p{", "/p}", "/p#c", "/p\nx", "/p\x00"} {
		if _, err := expandNginxSnippet(snippet, bad, "pub", "fpm"); err == nil {
			t.Errorf("path %q accepted", bad)
		}
	}
	if _, err := expandNginxSnippet(snippet, "/p", "pub", "fpm;deny all"); err == nil {
		t.Error("hostile fpm container name accepted")
	}
	if _, err := expandNginxSnippet(snippet, "/p", "pub}x{", "fpm"); err == nil {
		t.Error("hostile public dir accepted")
	}
}

// A snippet dropped because a substituted value carries nginx syntax (here a
// project path with a metacharacter) must warn, not vanish silently.
func TestFrameworkNginxBlockWarnsOnDrop(t *testing.T) {
	snippet := "location /a/ {\n    root {{root}};\n}"

	var buf bytes.Buffer
	if got := frameworkNginxBlock(&buf, "magento", "shop.test", snippet, "/home/u/a;b", "pub", "fpm"); got != "" {
		t.Fatalf("hostile path should be dropped, got %q", got)
	}
	if !strings.Contains(buf.String(), "[WARN]") || !strings.Contains(buf.String(), "shop.test") {
		t.Fatalf("drop was silent: %q", buf.String())
	}

	// A clean snippet expands and stays quiet.
	buf.Reset()
	if got := frameworkNginxBlock(&buf, "magento", "shop.test", snippet, "/home/u/a", "pub", "fpm"); got == "" {
		t.Fatal("valid snippet was dropped")
	}
	if buf.Len() != 0 {
		t.Fatalf("valid snippet warned: %q", buf.String())
	}
	// An empty snippet is the common case: no output, no block.
	buf.Reset()
	if got := frameworkNginxBlock(&buf, "laravel", "app.test", "  ", "/p", "public", "fpm"); got != "" || buf.Len() != 0 {
		t.Fatalf("empty snippet: got %q, warned %q", got, buf.String())
	}
}

// A dropped-snippet warning reaches the user once per process, so `lerd link`
// shows the reason while the watcher does not repeat it on every regeneration.
func TestEmitOnceDedupes(t *testing.T) {
	var buf bytes.Buffer
	msg := "[WARN] dropping magento nginx config for emitonce-unique.test: bad\n"
	emitOnce(&buf, msg)
	emitOnce(&buf, msg)
	if strings.Count(buf.String(), "emitonce-unique.test") != 1 {
		t.Fatalf("want one emission, got %q", buf.String())
	}
	// A different reason still surfaces; an empty message never emits.
	emitOnce(&buf, "[WARN] dropping magento nginx config for emitonce-unique.test: other\n")
	emitOnce(&buf, "")
	if strings.Count(buf.String(), "emitonce-unique.test") != 2 {
		t.Fatalf("distinct reason or empty msg mishandled: %q", buf.String())
	}
}

// The framework block must land inside the server block and BEFORE the generic
// `location /` and `location ~ \.php$`, since nginx picks the first matching
// regex location in declaration order.
func TestVhostRendersFrameworkNginxBeforeGenericLocations(t *testing.T) {
	data := VhostData{
		Domain:         "shop.test",
		ServerNames:    "shop.test",
		Path:           "/home/u/shop",
		PublicDir:      "pub",
		FPMContainer:   "lerd-php84-fpm",
		RequestTimeout: 60,
		FrameworkNginx: "    location ~* ^/setup($|/) {\n        root /home/u/shop;\n    }",
	}
	out := renderVhostForTest(t, "vhost.conf.tmpl", data)

	iSetup := strings.Index(out, "^/setup($|/)")
	iRoot := strings.Index(out, `root "/home/u/shop/pub";`)
	iSlash := strings.Index(out, "location / {")
	iPHP := strings.Index(out, `location ~ \.php$`)
	iClose := strings.LastIndex(out, "}")

	if iSetup < 0 {
		t.Fatalf("framework block missing:\n%s", out)
	}
	if !(iRoot < iSetup && iSetup < iSlash && iSetup < iPHP && iSetup < iClose) {
		t.Fatalf("bad ordering root=%d setup=%d slash=%d php=%d close=%d\n%s", iRoot, iSetup, iSlash, iPHP, iClose, out)
	}
}

func TestVhostOmitsFrameworkNginxWhenAbsent(t *testing.T) {
	for _, tmpl := range []string{"vhost.conf.tmpl", "vhost-ssl.conf.tmpl"} {
		data := VhostData{
			Domain: "a.test", ServerNames: "a.test", Path: "/p", PublicDir: "public",
			CertDomain: "a.test", FPMContainer: "f", RequestTimeout: 60,
		}
		out := renderVhostForTest(t, tmpl, data)
		if strings.Contains(out, "FrameworkNginx") {
			t.Fatalf("%s leaked field name", tmpl)
		}
		if !strings.Contains(out, "location / {") {
			t.Fatalf("%s missing generic location:\n%s", tmpl, out)
		}
	}
}

func TestSSLVhostRendersFrameworkNginx(t *testing.T) {
	data := VhostData{
		Domain: "shop.test", ServerNames: "shop.test", Path: "/home/u/shop", PublicDir: "pub",
		CertDomain: "shop.test", FPMContainer: "f", RequestTimeout: 60,
		FrameworkNginx: "    location /media/ {\n        try_files $uri /get.php;\n    }",
	}
	out := renderVhostForTest(t, "vhost-ssl.conf.tmpl", data)
	iMedia := strings.Index(out, "location /media/ {")
	iSlash := strings.Index(out, "location / {")
	if iMedia < 0 || iMedia > iSlash {
		t.Fatalf("media=%d slash=%d\n%s", iMedia, iSlash, out)
	}
}

// A framework's fastcgi params reach PHP as server variables, which is how a
// Testbench skeleton finds the package it serves: {{root}} expands to the
// quoted project path, and the params come out sorted so a regeneration is stable.
func TestFrameworkFastCGIParams(t *testing.T) {
	var buf bytes.Buffer
	got := frameworkFastCGIParams(&buf, "testbench", "pkg.test", map[string]string{
		"TESTBENCH_WORKING_PATH": "{{root}}",
		"APP_PUBLIC":             "{{public}}",
	}, "/home/u/my pkg", "vendor/laravel/public")
	want := "        fastcgi_param APP_PUBLIC \"/home/u/my pkg/vendor/laravel/public\";\n" +
		"        fastcgi_param TESTBENCH_WORKING_PATH \"/home/u/my pkg\";\n"
	if got != want || buf.Len() != 0 {
		t.Fatalf("got %q (warned %q), want %q", got, buf.String(), want)
	}
}

// A param that could break out of its directive is dropped with a warning, and
// the rest still render.
func TestFrameworkFastCGIParamsDropsHostileEntries(t *testing.T) {
	for name, tc := range map[string]struct {
		params map[string]string
		path   string
	}{
		"bad name":            {map[string]string{"A;B": "x"}, "/p"},
		"bad value":           {map[string]string{"A": "x;y"}, "/p"},
		"unknown placeholder": {map[string]string{"A": "{{roots}}"}, "/p"},
		"nginx variable":      {map[string]string{"A": "$price"}, "/p"},
		"hostile path":        {map[string]string{"A": "{{root}}"}, "/p;q"},
	} {
		var buf bytes.Buffer
		params := map[string]string{"OK": "1"}
		for k, v := range tc.params {
			params[k] = v
		}
		got := frameworkFastCGIParams(&buf, "fw", "a.test", params, tc.path, "public")
		if got != "        fastcgi_param OK \"1\";\n" {
			t.Errorf("%s: got %q", name, got)
		}
		if !strings.Contains(buf.String(), "[WARN]") || !strings.Contains(buf.String(), "a.test") {
			t.Errorf("%s: drop was silent: %q", name, buf.String())
		}
	}
}

// The params belong in the PHP location, after lerd's own, on both vhosts.
func TestVhostRendersFrameworkFastCGIInPHPLocation(t *testing.T) {
	line := "        fastcgi_param TESTBENCH_WORKING_PATH \"/home/u/pkg\";\n"
	for _, tmpl := range []string{"vhost.conf.tmpl", "vhost-ssl.conf.tmpl"} {
		out := renderVhostForTest(t, tmpl, VhostData{
			Domain: "pkg.test", ServerNames: "pkg.test", Path: "/home/u/pkg", PublicDir: "public",
			CertDomain: "pkg.test", FPMContainer: "f", RequestTimeout: 60, LerdSite: "pkg",
			FrameworkFastCGI: line,
		})
		iPHP := strings.Index(out, `location ~ \.php$`)
		iSite := strings.Index(out, "fastcgi_param LERD_SITE pkg;")
		iParam := strings.Index(out, strings.TrimSpace(line))
		iInclude := strings.Index(out, "custom.d/pkg.test.location.conf")
		if iParam < 0 || !(iPHP < iSite && iSite < iParam && iParam < iInclude) {
			t.Fatalf("%s: php=%d site=%d param=%d include=%d\n%s", tmpl, iPHP, iSite, iParam, iInclude, out)
		}
	}
}
