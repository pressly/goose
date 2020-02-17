[1mdiff --git a/creds/creds.go b/creds/creds.go[m
[1mindex 648ec17..1df08f1 100644[m
[1m--- a/creds/creds.go[m
[1m+++ b/creds/creds.go[m
[36m@@ -3,6 +3,7 @@[m [mpackage creds[m
 import ([m
 	"errors"[m
 	"fmt"[m
[32m+[m	[32m"github.com/apex/log"[m
 	"github.com/geniusmonkey/gander/env"[m
 	"github.com/geniusmonkey/gander/project"[m
 	"github.com/spf13/afero"[m
[36m@@ -14,6 +15,8 @@[m [mimport ([m
 var fs = afero.NewOsFs()[m
 var IsNotExist = errors.New("credentials do not exist")[m
 [m
[32m+[m[32mconst dirName = "gander"[m
[32m+[m
 type projectCreds map[string]Credentials[m
 [m
 type Credentials struct {[m
[36m@@ -34,19 +37,19 @@[m [mfunc Save(proj project.Project, environment env.Environment, credentials Credent[m
 		return err[m
 	}[m
 [m
[31m-	ganderCfg := path.Join(usrDir, "gander")[m
[32m+[m	[32mganderCfg := path.Join(usrDir, dirName)[m
 	credPath := path.Join(ganderCfg, proj.Name)[m
 [m
 	var file afero.File[m
[31m-	if ok, err := exists(credPath); err != nil {[m
[32m+[m	[32mif exists, err := afero.Exists(fs, credPath); err != nil {[m
 		return err[m
[31m-	} else if ok {[m
[31m-		file, err = os.Open(credPath)[m
[32m+[m	[32m} else if exists {[m
[32m+[m		[32mfile, err = fs.Open(credPath)[m
 	} else {[m
 		if err := fs.MkdirAll(ganderCfg, os.ModeDir|os.ModePerm); err != nil {[m
 			return fmt.Errorf("unable to create config directory, %w", err)[m
 		}[m
[31m-		file, err = os.Create(credPath)[m
[32m+[m		[32mfile, err = fs.Create(credPath)[m
 	}[m
 [m
 	if err != nil {[m
[36m@@ -76,10 +79,11 @@[m [mfunc loadProCreds(proj project.Project) (projectCreds, error) {[m
 		return pc, err[m
 	}[m
 [m
[31m-	credPath := path.Join(usrDir, "gander", proj.Name)[m
[31m-	if ok, err := exists(credPath); err != nil {[m
[32m+[m	[32mcredPath := path.Join(usrDir, dirName, proj.Name)[m
[32m+[m	[32mif exists, err := afero.Exists(fs, credPath); err != nil {[m
 		return pc, err[m
[31m-	} else if ok {[m
[32m+[m	[32m} else if exists {[m
[32m+[m		[32mlog.Debugf("using credentials file found at $s", credPath)[m
 		file, err := fs.Open(credPath)[m
 		if err != nil {[m
 			return pc, err[m
[36m@@ -87,17 +91,8 @@[m [mfunc loadProCreds(proj project.Project) (projectCreds, error) {[m
 		err = yaml.NewDecoder(file).Decode(&pc)[m
 		return pc, err[m
 	} else {[m
[32m+[m		[32mlog.Debugf("credential file %s does not exist returning empty config", credPath)[m
 		return pc, nil[m
 	}[m
 }[m
 [m
[31m-func exists(path string) (bool, error) {[m
[31m-	_, err := os.Stat(path)[m
[31m-	if err == nil {[m
[31m-		return true, nil[m
[31m-	}[m
[31m-	if os.IsNotExist(err) {[m
[31m-		return false, nil[m
[31m-	}[m
[31m-	return true, err[m
[31m-}[m
[1mdiff --git a/log.go b/log.go[m
[1mindex 6b9340b..92a2bbb 100644[m
[1m--- a/log.go[m
[1m+++ b/log.go[m
[36m@@ -19,7 +19,7 @@[m [mfunc init() {[m
 			Writer:  os.Stdout,[m
 			Padding: 0,[m
 		},[m
[31m-		Level:   std.InfoLevel,[m
[32m+[m		[32mLevel:   std.DebugLevel,[m
 	}[m
 }[m
 [m
[1mdiff --git a/project/project.go b/project/project.go[m
[1mindex 2b4053c..5fed3db 100644[m
[1m--- a/project/project.go[m
[1m+++ b/project/project.go[m
[36m@@ -13,6 +13,8 @@[m [mvar fs afero.Fs[m
 var ErrProjectExists = errors.New("existing project")[m
 var IsNotExists = errors.New("not a gander project dir")[m
 [m
[32m+[m[32mconst dirName = ".gander"[m
[32m+[m
 func init() {[m
 	fs = afero.NewOsFs()[m
 }[m
[36m@@ -30,7 +32,7 @@[m [mfunc (p Project) MigrationDir() string {[m
 }[m
 [m
 func Init(dir string, project Project) error {[m
[31m-	ganderDir := path.Join(dir, ".gander")[m
[32m+[m	[32mganderDir := path.Join(dir, dirName)[m
 	_, err := fs.Stat(ganderDir)[m
 	if err == nil {[m
 		return ErrProjectExists[m
[36m@@ -56,7 +58,7 @@[m [mfunc Init(dir string, project Project) error {[m
 }[m
 [m
 func Get(dir string) (*Project, error) {[m
[31m-	ganderDir := path.Join(dir, ".gander", "project.yaml")[m
[32m+[m	[32mganderDir := path.Join(dir, dirName, "project.yaml")[m
 	file, err := fs.Open(ganderDir)[m
 	if err != nil {[m
 		return nil, IsNotExists[m
