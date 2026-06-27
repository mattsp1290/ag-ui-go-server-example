module github.com/mattsp1290/ag-ui-go-server-example

go 1.26.3

// TODO: these replace directives point at local sibling checkouts for the local
// proof. Before sharing/publishing this module, pin to published versions (or
// vendor) and remove the absolute-path replaces — the module won't build for
// anyone without these exact paths.
replace github.com/ag-ui-protocol/ag-ui/sdks/community/go => /Users/punk1290/git/ag-ui/sdks/community/go

replace github.com/mattsp1290/eino-providers => /Users/punk1290/git/eino-providers

replace github.com/mattsp1290/codex-auth-go => /Users/punk1290/git/codex-auth-go

replace github.com/mattsp1290/eino-tools => /Users/punk1290/git/eino-tools

require (
	github.com/ag-ui-protocol/ag-ui/sdks/community/go v0.0.0-20260624151131-d2049debabd9
	github.com/cloudwego/eino v0.9.2
	github.com/cloudwego/eino-ext/components/model/openai v0.1.13
	github.com/eino-contrib/jsonschema v1.0.3
	github.com/evanphx/json-patch v0.5.2
	github.com/gofiber/fiber/v3 v3.3.0
	github.com/mattsp1290/eino-agui v0.1.1
	github.com/mattsp1290/eino-providers v0.0.0-00010101000000-000000000000
	github.com/mattsp1290/eino-tools v0.0.0-00010101000000-000000000000
)

require (
	cloud.google.com/go v0.116.0 // indirect
	cloud.google.com/go/auth v0.9.3 // indirect
	cloud.google.com/go/compute/metadata v0.5.0 // indirect
	github.com/andybalholm/brotli v1.2.1 // indirect
	github.com/bahlo/generic-list-go v0.2.0 // indirect
	github.com/buger/jsonparser v1.1.1 // indirect
	github.com/bytedance/gopkg v0.1.3 // indirect
	github.com/bytedance/sonic v1.15.0 // indirect
	github.com/bytedance/sonic/loader v0.5.0 // indirect
	github.com/cloudwego/base64x v0.1.6 // indirect
	github.com/cloudwego/eino-ext/libs/acl/openai v0.1.17 // indirect
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/gofiber/schema v1.7.1 // indirect
	github.com/gofiber/utils/v2 v2.0.6 // indirect
	github.com/gofrs/flock v0.13.0 // indirect
	github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da // indirect
	github.com/google/go-cmp v0.7.0 // indirect
	github.com/google/s2a-go v0.1.8 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/googleapis/enterprise-certificate-proxy v0.3.4 // indirect
	github.com/goph/emperror v0.17.2 // indirect
	github.com/gorilla/websocket v1.5.3 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/klauspost/compress v1.18.6 // indirect
	github.com/klauspost/cpuid/v2 v2.2.9 // indirect
	github.com/mailru/easyjson v0.7.7 // indirect
	github.com/mattn/go-colorable v0.1.14 // indirect
	github.com/mattn/go-isatty v0.0.22 // indirect
	github.com/mattsp1290/codex-auth-go v0.1.0 // indirect
	github.com/meguminnnnnnnnn/go-openai v0.1.2 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/nikolalohinski/gonja v1.5.3 // indirect
	github.com/pelletier/go-toml/v2 v2.2.2 // indirect
	github.com/philhofer/fwd v1.2.0 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/rogpeppe/go-internal v1.15.0 // indirect
	github.com/sirupsen/logrus v1.9.3 // indirect
	github.com/slongfield/pyfmt v0.0.0-20220222012616-ea85ff4c361f // indirect
	github.com/tinylib/msgp v1.6.4 // indirect
	github.com/twitchyliquid64/golang-asm v0.15.1 // indirect
	github.com/valyala/bytebufferpool v1.0.0 // indirect
	github.com/valyala/fasthttp v1.71.0 // indirect
	github.com/wk8/go-ordered-map/v2 v2.1.8 // indirect
	github.com/yargevad/filepathx v1.0.0 // indirect
	go.opencensus.io v0.24.0 // indirect
	golang.org/x/arch v0.12.0 // indirect
	golang.org/x/crypto v0.51.0 // indirect
	golang.org/x/exp v0.0.0-20250218142911-aa4b98e5adaa // indirect
	golang.org/x/net v0.54.0 // indirect
	golang.org/x/sync v0.20.0 // indirect
	golang.org/x/sys v0.44.0 // indirect
	golang.org/x/text v0.37.0 // indirect
	google.golang.org/genai v1.55.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20240903143218-8af14fe29dc1 // indirect
	google.golang.org/grpc v1.66.2 // indirect
	google.golang.org/protobuf v1.34.2 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
