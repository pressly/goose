package cli

var flagLookup = map[string]flagUsage{
	"allow-missing": {
		short:         "Allow missing, out-of-order, migrations",
		defaultOption: "",
	},
	"dbstring": {
		short:         "Database connection string",
		defaultOption: "",
	},
	"dir": {
		short:         "Directory with migration files",
		defaultOption: "./migrations",
	},
	"exclude": {
		short:         "Exclude migrations by filename, comma separated",
		defaultOption: "",
	},
	"help": {
		short:         "Display help",
		defaultOption: "",
	},
	"json": {
		short:         "Format output as JSON",
		defaultOption: "",
	},
	"lock-mode": {
		short:            "Set lock mode",
		availableOptions: []string{"none", "advisory-session"},
		defaultOption:    "none",
	},
	"no-versioning": {
		short:         "Do not store version info in database, just run migrations",
		defaultOption: "",
	},
	"table": {
		short:         "Database table name to store version info",
		defaultOption: "goose_db_version",
	},
	"v": {
		short:         "Turn on verbose mode",
		defaultOption: "",
	},
	"version": {
		short:         "Display goose cli version",
		defaultOption: "",
	},
}
