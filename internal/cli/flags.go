package cli

type flagUsage struct {
	// short is the short description of the flag
	short string
	// defaultOption is the default value of the flag
	defaultOption string
	// availableOptions is a list of available options for the flag
	availableOptions []string
}

var flagLookup = map[string]flagUsage{
	"allow-missing": {short: "Allow missing, out-of-order, migrations", defaultOption: ""},
	"dbstring":      {short: "Database connection string", defaultOption: ""},
	"dir":           {short: "Directory with migration files", defaultOption: ""},
	"exclude":       {short: "Exclude migrations by filename, comma separated", defaultOption: ""},
	"help":          {short: "Display help", defaultOption: ""},
	"json":          {short: "Format output as JSON", defaultOption: ""},
	"lock-mode":     {short: "Set lock mode", availableOptions: []string{"none", "advisory-session"}, defaultOption: "none"},
	"no-color":      {short: "Disable color output", defaultOption: ""},
	"no-tx":         {short: "Do not wrap migration in a transaction", defaultOption: ""},
	"no-versioning": {short: "Do not store version info in database, just run migrations", defaultOption: ""},
	"s":             {short: "Sequentially number migrations", defaultOption: ""},
	"table":         {short: "Database table name to store version info", defaultOption: DefaultTableName},
	"v":             {short: "Turn on verbose mode", defaultOption: ""},
	"version":       {short: "Display goose cli version", defaultOption: ""},
}
