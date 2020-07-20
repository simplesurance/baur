package baur

import (
	"fmt"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/pkg/errors"
	"github.com/rs/xid"

	"github.com/simplesurance/baur/cfg"
	"github.com/simplesurance/baur/digest"
	"github.com/simplesurance/baur/digest/sha384"
	"github.com/simplesurance/baur/log"
	"github.com/simplesurance/baur/resolve/gitpath"
	"github.com/simplesurance/baur/resolve/glob"
	"github.com/simplesurance/baur/resolve/gosource"
	"github.com/simplesurance/baur/upload/scheduler"
)

// App represents an application
type App struct {
	RelPath          string
	Path             string
	Name             string
	BuildCmd         string
	Repository       *Repository
	Outputs          []BuildOutput
	totalInputDigest *digest.Digest

	UnresolvedInputs []*cfg.BuildInput
	buildInputs      []*File
}

func replaceUUIDvar(in string) string {
	return strings.Replace(in, "$UUID", xid.New().String(), -1)
}

func replaceROOTvar(in string, r *Repository) string {
	return strings.Replace(in, "$ROOT", r.Path, -1)
}

func replaceAppNameVar(in, appName string) string {
	return strings.Replace(in, "$APPNAME", appName, -1)
}

func replaceGitCommitVar(in string, r *Repository) (string, error) {
	commitID, err := r.GitCommitID()
	if err != nil {
		return "", err
	}

	return strings.Replace(in, "$GITCOMMIT", commitID, -1), nil
}

func (a *App) addBuildOutput(buildOutput *cfg.BuildOutput) error {
	if err := a.addDockerBuildOutputs(buildOutput); err != nil {
		return errors.Wrap(err, "error in DockerImage section")
	}

	if err := a.addFileOutputs(buildOutput); err != nil {
		return errors.Wrap(err, "error in File section")
	}

	return nil
}

func (a *App) addDockerBuildOutputs(buildOutput *cfg.BuildOutput) error {
	for _, di := range buildOutput.DockerImage {
		tag, err := replaceGitCommitVar(di.RegistryUpload.Tag, a.Repository)
		if err != nil {
			return errors.Wrap(err, "replacing $GITCOMMIT in tag failed")
		}

		tag = replaceUUIDvar(tag)
		repository := replaceAppNameVar(di.RegistryUpload.Repository, a.Name)

		a.Outputs = append(a.Outputs, &DockerArtifact{
			ImageIDFile: path.Join(a.Path, replaceAppNameVar(di.IDFile, a.Name)),
			Tag:         tag,
			Repository:  repository,
			Registry:    di.RegistryUpload.Registry,
		})
	}

	return nil
}

func (a *App) addFileOutputs(buildOutput *cfg.BuildOutput) error {
	for _, f := range buildOutput.File {
		filePath := replaceAppNameVar(f.Path, a.Name)
		if !f.S3Upload.IsEmpty() {
			destFile, err := replaceGitCommitVar(f.S3Upload.DestFile, a.Repository)
			if err != nil {
				return errors.Wrap(err, "replacing $GITCOMMIT in dest_file failed")
			}

			destFile = replaceUUIDvar(replaceAppNameVar(destFile, a.Name))
			s3Bucket := replaceAppNameVar(f.S3Upload.Bucket, a.Name)
			url := "s3://" + s3Bucket + "/" + destFile

			src := path.Join(a.Path, filePath)

			a.Outputs = append(a.Outputs, &FileArtifact{
				RelPath:   path.Join(a.RelPath, filePath),
				Path:      src,
				DestFile:  destFile,
				UploadURL: url,
				uploadJob: &scheduler.S3Job{
					DestURL:  url,
					FilePath: src,
				},
			})
		}

		if !f.FileCopy.IsEmpty() {
			dest, err := replaceGitCommitVar(f.FileCopy.Path, a.Repository)
			if err != nil {
				return errors.Wrap(err, "replacing $GITCOMMIT in path failed")
			}

			dest = replaceUUIDvar(replaceAppNameVar(dest, a.Name))
			src := path.Join(a.Path, filePath)

			a.Outputs = append(a.Outputs, &FileArtifact{
				RelPath:   path.Join(a.RelPath, filePath),
				Path:      src,
				DestFile:  dest,
				UploadURL: dest,
				uploadJob: &scheduler.FileCopyJob{
					Src: src,
					Dst: dest,
				},
			})

		}
	}

	return nil
}

func (a *App) include(inc *cfg.Include) error {
	a.UnresolvedInputs = append(a.UnresolvedInputs, &inc.BuildInput)

	return a.addBuildOutput(&inc.BuildOutput)
}

func (a *App) loadIncludes(appCfg *cfg.App) error {
	for _, includePath := range appCfg.Build.Includes {
		path := replaceROOTvar(includePath, a.Repository)
		if !filepath.IsAbs(path) {
			path = filepath.Join(a.Path, path)
		}

		inc, err := a.Repository.includeCache.load(path)
		if err != nil {
			return errors.Wrapf(err, "loading include '%s' failed", includePath)
		}

		err = a.include(inc)
		if err != nil {
			return errors.Wrapf(err, "including '%s' failed", includePath)
		}
	}

	return nil
}

func (a *App) addCfgsToBuildInputs(appCfg *cfg.App) {
	buildInput := cfg.BuildInput{}
	buildInput.Files.Paths = append(buildInput.Files.Paths, AppCfgFile)
	buildInput.Files.Paths = append(buildInput.Files.Paths, appCfg.Build.Includes...)

	a.UnresolvedInputs = append(a.UnresolvedInputs, &buildInput)
}

// NewApp reads the configuration file and returns a new App
func NewApp(repository *Repository, cfgPath string) (*App, error) {
	appCfg, err := cfg.AppFromFile(cfgPath)
	if err != nil {
		return nil, errors.Wrapf(err,
			"reading application config %s failed", cfgPath)
	}

	err = appCfg.Validate()
	if err != nil {
		return nil, errors.Wrapf(err,
			"validating application config %s failed",
			cfgPath)
	}

	appAbsPath := path.Dir(cfgPath)
	appRelPath, err := filepath.Rel(repository.Path, appAbsPath)
	if err != nil {
		return nil, errors.Wrapf(err, "%s: resolving repository relative application path failed", appCfg.Name)
	}

	cmd := strings.TrimSpace(appCfg.Build.Command)
	cmd = replaceROOTvar(cmd, repository)
	cmd = replaceAppNameVar(cmd, appCfg.Name)

	app := App{
		Repository: repository,
		Path:       path.Dir(cfgPath),
		RelPath:    appRelPath,
		Name:       appCfg.Name,
		BuildCmd:   cmd,
	}

	err = app.addBuildOutput(&appCfg.Build.Output)
	if err != nil {
		return nil, errors.Wrapf(err, "%s: processing Build.Output section failed", app.Name)
	}

	app.UnresolvedInputs = []*cfg.BuildInput{&appCfg.Build.Input}
	app.addCfgsToBuildInputs(appCfg)

	err = app.loadIncludes(appCfg)
	if err != nil {
		return nil, errors.Wrapf(err, "%s: processing application config failed failed", app.Name)
	}

	return &app, nil
}

// String returns the string representation of an app
func (a *App) String() string {
	return a.Name
}

func (a *App) pathsToUniqFiles(paths []string) ([]*File, error) {
	dedupMap := make(map[string]struct{}, len(paths))
	res := make([]*File, 0, len(paths))

	for _, path := range paths {
		if _, exist := dedupMap[path]; exist {
			log.Debugf("%s: removed duplicate Build Input '%s'", a.Name, path)
			continue
		}
		dedupMap[path] = struct{}{}

		relPath, err := filepath.Rel(a.Repository.Path, path)
		if err != nil {
			return nil, errors.Wrapf(err, "resolving relative path to '%s' from '%s' failed", path, a.Repository.Path)
		}

		// TODO: should resolving the relative path be done in
		// Newfile() instead?
		res = append(res, NewFile(a.Repository.Path, relPath))
	}

	return res, nil
}

func (a *App) resolveGlobFileInputs() ([]string, error) {
	var res []string

	for _, bi := range a.UnresolvedInputs {
		for _, globPath := range bi.Files.Paths {
			if strings.HasPrefix(globPath, "$ROOT") {
				globPath = filepath.Clean(replaceROOTvar(globPath, a.Repository))
			}

			if !filepath.IsAbs(globPath) {
				globPath = filepath.Join(a.Path, globPath)
			}

			resolver := glob.NewResolver(globPath)
			paths, err := resolver.Resolve()
			if err != nil {
				return nil, errors.Wrap(err, globPath)
			}

			if len(paths) == 0 {
				return nil, fmt.Errorf("'%s' matched 0 files", globPath)
			}

			res = append(res, paths...)
		}
	}

	return res, nil
}

func (a *App) resolveGitFileInputs() ([]string, error) {
	var res []string

	for _, bi := range a.UnresolvedInputs {
		if len(bi.GitFiles.Paths) == 0 {
			continue
		}

		paths := make([]string, 0, len(bi.GitFiles.Paths))
		for _, path := range bi.GitFiles.Paths {
			if !strings.HasPrefix(path, "$ROOT") {
				paths = append(paths, path)
				continue
			}

			absPath := replaceROOTvar(path, a.Repository)
			relPath, err := filepath.Rel(a.Path, absPath)
			if err != nil {
				return nil, err
			}

			paths = append(paths, relPath)
		}

		resolver := gitpath.NewResolver(a.Path, paths...)
		paths, err := resolver.Resolve()
		if err != nil {
			return nil, err
		}

		if len(paths) == 0 {
			return nil, fmt.Errorf("'%s' matched 0 files", strings.Join(paths, ", "))
		}
		res = append(res, paths...)
	}

	return res, nil
}

func (a *App) resolveGoSrcInputs() ([]string, error) {
	var res []string

	for _, bi := range a.UnresolvedInputs {
		if len(bi.GolangSources.Paths) == 0 {
			continue
		}

		absGoSourcePaths := make([]string, 0, len(bi.GolangSources.Paths))
		for _, relGosrcpath := range bi.GolangSources.Paths {
			absPath := path.Join(a.Path, relGosrcpath)
			absGoSourcePaths = append(absGoSourcePaths, absPath)
		}

		goSrcEnv := make([]string, 0, len(bi.GolangSources.Environment))
		for _, val := range bi.GolangSources.Environment {
			goSrcEnv = append(goSrcEnv, path.Clean(replaceROOTvar(val, a.Repository)))
		}

		resolver := gosource.NewResolver(log.Debugf, goSrcEnv, absGoSourcePaths...)
		paths, err := resolver.Resolve()
		if err != nil {
			return nil, err
		}

		if len(paths) == 0 {
			return nil, fmt.Errorf("'%s' matched 0 files", strings.Join(paths, ", "))
		}

		res = append(res, paths...)
	}

	return res, nil
}

func (a *App) resolveBuildInputPaths() ([]string, error) {
	globPaths, err := a.resolveGlobFileInputs()
	if err != nil {
		return nil, errors.Wrapf(err, "resolving File BuildInputs failed")
	}

	gitPaths, err := a.resolveGitFileInputs()
	if err != nil {
		return nil, errors.Wrapf(err, "resolving GitFile BuildInputs failed")
	}

	goSrcPaths, err := a.resolveGoSrcInputs()
	if err != nil {
		return nil, errors.Wrapf(err, "resolving GoLangSources BuildInputs failed")
	}

	paths := make([]string, 0, len(globPaths)+len(gitPaths)+len(goSrcPaths))
	paths = append(paths, globPaths...)
	paths = append(paths, gitPaths...)
	paths = append(paths, goSrcPaths...)

	return paths, nil
}

// HasBuildInputs returns true if BuildInputs are defined for the app
func (a *App) HasBuildInputs() bool {
	for _, bi := range a.UnresolvedInputs {
		if len(bi.Files.Paths) != 0 {
			return true
		}

		if len(bi.GitFiles.Paths) != 0 {
			return true
		}

		if len(bi.GolangSources.Paths) != 0 {
			return true
		}
	}

	return false
}

// BuildInputs resolves all build inputs of the app.
// The BuildInputs are deduplicates before they are returned.
// If one more resolved path does not match a file an error is generated.
// If not build inputs are defined, an empty slice and no error is returned.
// If the function is called the first time, the BuildInputPaths are resolved
// and stored. On following calls the stored BuildInputs are returned.
func (a *App) BuildInputs() ([]*File, error) {
	if a.buildInputs != nil {
		return a.buildInputs, nil
	}

	paths, err := a.resolveBuildInputPaths()
	if err != nil {
		return nil, err
	}

	a.buildInputs, err = a.pathsToUniqFiles(paths)
	if err != nil {
		return nil, err
	}

	return a.buildInputs, nil
}

// TotalInputDigest returns the total input digest that is calculated over all
// input sources. The calculation is only done on the 1. call on following calls
// the stored digest is returned
func (a *App) TotalInputDigest() (digest.Digest, error) {
	if a.totalInputDigest != nil {
		return *a.totalInputDigest, nil
	}

	buildInputs, err := a.BuildInputs()
	if err != nil {
		return digest.Digest{}, err
	}

	digests := make([]*digest.Digest, 0, len(buildInputs))
	for _, bi := range buildInputs {
		d, err := bi.Digest()
		if err != nil {
			return digest.Digest{}, errors.Wrapf(err, "calculating input digest of %q failed", bi)
		}

		digests = append(digests, &d)
	}

	totalDigest, err := sha384.Sum(digests)
	if err != nil {
		return digest.Digest{}, errors.Wrap(err, "calculating total input digest")
	}

	a.totalInputDigest = totalDigest

	return *a.totalInputDigest, nil
}

// SortAppsByName sorts the apps in the slice by Name
func SortAppsByName(apps []*App) {
	sort.Slice(apps, func(i int, j int) bool {
		return apps[i].Name < apps[j].Name
	})
}
