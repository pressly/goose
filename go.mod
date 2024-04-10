module github.com/pressly/goose/v3

go 1.20

require (
	github.com/ClickHouse/clickhouse-go/v2 v2.17.1
	github.com/charmbracelet/lipgloss v0.10.0
	github.com/go-sql-driver/mysql v1.8.0
	github.com/jackc/pgx/v5 v5.5.5
	github.com/mfridman/interpolate v0.0.2
	github.com/microsoft/go-mssqldb v1.7.0
	github.com/peterbourgon/ff/v4 v4.0.0-alpha.4
	github.com/sethvargo/go-retry v0.2.4
	github.com/stretchr/testify v1.9.0
	github.com/tursodatabase/libsql-client-go v0.0.0-20240220085343-4ae0eb9d0898
	github.com/vertica/vertica-sql-go v1.3.3
	github.com/xo/dburl v0.22.0
	github.com/ydb-platform/ydb-go-sdk/v3 v3.55.1
	github.com/ziutek/mymysql v1.5.4
	go.uber.org/multierr v1.11.0
	golang.org/x/sync v0.6.0
	modernc.org/sqlite v1.29.5
)

require (
	filippo.io/edwards25519 v1.1.0 // indirect
	github.com/ClickHouse/ch-go v0.58.2 // indirect
	github.com/andybalholm/brotli v1.0.6 // indirect
	github.com/antlr/antlr4/runtime/Go/antlr/v4 v4.0.0-20230512164433-5d1fd1a340c9 // indirect
	github.com/aymanbagabas/go-osc52/v2 v2.0.1 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/elastic/go-sysinfo v1.11.2 // indirect
	github.com/elastic/go-windows v1.0.1 // indirect
	github.com/go-faster/city v1.0.1 // indirect
	github.com/go-faster/errors v0.6.1 // indirect
	github.com/golang-jwt/jwt/v4 v4.5.0 // indirect
	github.com/golang-sql/civil v0.0.0-20220223132316-b832511892a9 // indirect
	github.com/golang-sql/sqlexp v0.1.0 // indirect
	github.com/golang/protobuf v1.5.3 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/hashicorp/golang-lru/v2 v2.0.7 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20221227161230-091c0ba34f0a // indirect
	github.com/jackc/puddle/v2 v2.2.1 // indirect
	github.com/joeshaw/multierror v0.0.0-20140124173710-69b34d4ec901 // indirect
	github.com/jonboulle/clockwork v0.4.0 // indirect
	github.com/klauspost/compress v1.17.2 // indirect
	github.com/libsql/sqlite-antlr4-parser v0.0.0-20230802215326-5cb5bb604475 // indirect
	github.com/lucasb-eyer/go-colorful v1.2.0 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/mattn/go-runewidth v0.0.15 // indirect
	github.com/muesli/reflow v0.3.0 // indirect
	github.com/muesli/termenv v0.15.2 // indirect
	github.com/ncruces/go-strftime v0.1.9 // indirect
	github.com/paulmach/orb v0.10.0 // indirect
	github.com/pierrec/lz4/v4 v4.1.18 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/prometheus/procfs v0.12.0 // indirect
	github.com/remyoudompheng/bigfft v0.0.0-20230129092748-24d4a6f8daec // indirect
	github.com/rivo/uniseg v0.4.7 // indirect
	github.com/segmentio/asm v1.2.0 // indirect
	github.com/shopspring/decimal v1.3.1 // indirect
	github.com/ydb-platform/ydb-go-genproto v0.0.0-20240126124512-dbb0e1720dbf // indirect
	go.opentelemetry.io/otel v1.20.0 // indirect
	go.opentelemetry.io/otel/trace v1.20.0 // indirect
	golang.org/x/crypto v0.18.0 // indirect
	golang.org/x/exp v0.0.0-20231108232855-2478ac86f678 // indirect
	golang.org/x/net v0.20.0 // indirect
	golang.org/x/sys v0.16.0 // indirect
	golang.org/x/text v0.14.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20231106174013-bbf56f31fb17 // indirect
	google.golang.org/grpc v1.59.0 // indirect
	google.golang.org/protobuf v1.33.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	howett.net/plist v1.0.0 // indirect
	modernc.org/gc/v3 v3.0.0-20240107210532-573471604cb6 // indirect
	modernc.org/libc v1.41.0 // indirect
	modernc.org/mathutil v1.6.0 // indirect
	modernc.org/memory v1.7.2 // indirect
	modernc.org/strutil v1.2.0 // indirect
	modernc.org/token v1.1.0 // indirect
	nhooyr.io/websocket v1.8.7 // indirect
)

retract (
	v3.12.2 // Invalid module reference
	v3.12.1 // Invalid module reference
	v3.12.0 // Invalid module reference
)
