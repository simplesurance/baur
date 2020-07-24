package cfg

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/simplesurance/baur/v1/cfg/resolver"
)

type LogFn func(format string, v ...interface{})

// IncludeDB loads and stores include config files.
// It's methods are not concurrency-safe.
type IncludeDB struct {
	logf LogFn

	// the first maps use the absolute path to the include file as key, the second maps use the include ID as key
	inputs  map[string]map[string]*InputInclude
	outputs map[string]map[string]*OutputInclude
	tasks   map[string]map[string]*TaskInclude
}

// ErrIncludeIDNotFound describes that an include with a specific does not exist in an include file.
var ErrIncludeIDNotFound = errors.New("id not found in include file")

func NewIncludeDB(logf LogFn) *IncludeDB {
	if logf == nil {
		logf = func(_ string, _ ...interface{}) {}

	}
	return &IncludeDB{
		inputs:  map[string]map[string]*InputInclude{},
		outputs: map[string]map[string]*OutputInclude{},
		tasks:   map[string]map[string]*TaskInclude{},
		logf:    logf,
	}
}

// LoadTaskInclude loads the TaskInclude with the given ID.
// If the include was loaded before it is retrieved from the db.
// If it wasn't the include file is loaded and added to the db.
// If the file exist but does not have an include with the specified ID,
// IncludeIDNotFoundError is returned.
func (db *IncludeDB) LoadTaskInclude(resolver resolver.Resolver, workingDir, includeSpec string) (*TaskInclude, error) {
	absPath, id, err := db.parseIncludeSpec(resolver, workingDir, includeSpec)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", includeSpecifier(absPath, id), err)
	}

	if idMap, exist := db.tasks[absPath]; exist {
		if include, exist := idMap[id]; exist {
			return include, nil
		}

		return nil, ErrIncludeIDNotFound
	}

	if err := db.load(absPath, resolver); err != nil {
		return nil, err
	}

	include, exist := db.taskInclude(absPath, id)
	if !exist {
		return nil, ErrIncludeIDNotFound
	}

	return include, nil
}

// LoadInputInclude loads the InputInclude with the given ID.
// If the include was loaded before it is retrieved from the db.
// If it wasn't the include file is loaded and added to the db.
// If the file exist but does not have an include with the specified ID,
// IncludeIDNotFoundError is returned.
func (db *IncludeDB) LoadInputInclude(resolver resolver.Resolver, workingDir, includeSpec string) (*InputInclude, error) {
	absPath, id, err := db.parseIncludeSpec(resolver, workingDir, includeSpec)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", includeSpecifier(absPath, id), err)
	}

	if idMap, exist := db.inputs[absPath]; exist {
		if include, exist := idMap[id]; exist {
			return include, nil
		}

		return nil, ErrIncludeIDNotFound
	}

	if err := db.load(absPath, resolver); err != nil {
		return nil, err
	}

	include, exist := db.inputInclude(absPath, id)
	if !exist {
		return nil, ErrIncludeIDNotFound
	}

	return include, nil
}

// LoadOutputInclude loads the OutputInclude with the given ID.
// If the include was loaded before it is retrieved from the db.
// If it wasn't the include file is loaded and added to the db.
// If the file exist but does not have an include with the specified ID,
// IncludeIDNotFoundError is returned.
func (db *IncludeDB) LoadOutputInclude(resolver resolver.Resolver, workingDir, includeSpec string) (*OutputInclude, error) {
	absPath, id, err := db.parseIncludeSpec(resolver, workingDir, includeSpec)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", includeSpecifier(absPath, id), err)
	}

	if idMap, exist := db.outputs[absPath]; exist {
		if include, exist := idMap[id]; exist {
			return include, nil
		}

		return nil, ErrIncludeIDNotFound
	}

	if err := db.load(absPath, resolver); err != nil {
		return nil, err
	}

	include, exist := db.outputInclude(absPath, id)
	if !exist {
		return nil, ErrIncludeIDNotFound
	}

	return include, nil

}

// parseIncludeSpec splits the includeSpecifier to an absolute path and an include ID.
// If the path is not an absolute path after it was resolved, it is joined with the passed workingDir.
func (db *IncludeDB) parseIncludeSpec(resolver resolver.Resolver, workingDir, include string) (absPath, id string, err error) {
	spl := strings.Split(include, includeIDSep)
	if len(spl) != 2 {
		return "", "", errors.New("not a valid include specifier, does not contain exactly one '#' character")
	}

	relPath := spl[0]
	id = spl[1]

	path, err := resolver.Resolve(relPath)
	if err != nil {
		return "", "", fmt.Errorf("resolving variables failed: %w", err)
	}

	if !filepath.IsAbs(path) {
		path = filepath.Join(workingDir, relPath)
	}

	db.logf("includedb: resolved %q to path: %q, id: %q", include, path, id)

	return path, id, nil
}

// load loads the include file, resolves it's variables, validates it and adds it to the IncludeDB.
// Includes referenced in TaskIncludes a recursively loaded and included.
func (db *IncludeDB) load(path string, resolver resolver.Resolver) error {
	db.logf("includedb: loading %q", path)
	include, err := IncludeFromFile(path)
	if err != nil {
		// the error includes the path to the file
		if os.IsNotExist(err) {
			return err
		}
		return fmt.Errorf("%s: %w", path, err)
	}

	// Inputs and Outputs are load and indexed before the Tasks. This
	// allows to include inputs and outputs of the same file in the TaskInclude.

	if err := include.Input.Validate(); err != nil {
		return fmt.Errorf("validation failed: %w", FieldErrorWrap(err, "Input"))
	}

	for _, input := range include.Input {
		if err := db.inputOutputIncludeNotExist(path, input.IncludeID); err != nil {
			return err
		}

		db.addInputInclude(path, input)
	}

	if err := include.Output.Validate(); err != nil {
		return fmt.Errorf("validation failed: %w", FieldErrorWrap(err, "Output"))
	}

	for _, output := range include.Output {
		if err := db.inputOutputIncludeNotExist(path, output.IncludeID); err != nil {
			return err
		}

		db.addOutputInclude(path, output)
	}

	if err := include.Task.Merge(filepath.Dir(path), resolver, db); err != nil {
		return fmt.Errorf("merge failed: %w", FieldErrorWrap(err, "Task"))
	}

	if err := include.Task.Validate(); err != nil {
		return fmt.Errorf("validation failed: %w", FieldErrorWrap(err, "Task"))
	}

	for _, task := range include.Task {
		db.addTaskInclude(path, task)
	}

	return nil
}

func (db *IncludeDB) addTaskInclude(absPath string, include *TaskInclude) {
	idMap, exist := db.tasks[absPath]
	if !exist {
		idMap = map[string]*TaskInclude{}
		db.tasks[absPath] = idMap
	}

	idMap[include.IncludeID] = include
	db.logf("includedb: loaded include %q", includeSpecifier(absPath, include.IncludeID))
}

func (db *IncludeDB) addOutputInclude(absPath string, include *OutputInclude) {
	idMap, exist := db.outputs[absPath]
	if !exist {
		idMap = map[string]*OutputInclude{}
		db.outputs[absPath] = idMap
	}

	idMap[include.IncludeID] = include
	db.logf("includedb: loaded include %q", includeSpecifier(absPath, include.IncludeID))
}

func (db *IncludeDB) addInputInclude(absPath string, include *InputInclude) {
	idMap, exist := db.inputs[absPath]
	if !exist {
		idMap = map[string]*InputInclude{}
		db.inputs[absPath] = idMap
	}

	idMap[include.IncludeID] = include
	db.logf("includedb: loaded include %q", includeSpecifier(absPath, include.IncludeID))
}

func (db *IncludeDB) taskInclude(absPath, id string) (*TaskInclude, bool) {
	if idMap, exist := db.tasks[absPath]; exist {
		if include, exist := idMap[id]; exist {
			return include, true
		}
	}

	return nil, false
}

func (db *IncludeDB) inputInclude(absPath, id string) (*InputInclude, bool) {
	if idMap, exist := db.inputs[absPath]; exist {
		if include, exist := idMap[id]; exist {
			return include, true
		}
	}

	return nil, false
}

func (db *IncludeDB) outputInclude(absPath, id string) (*OutputInclude, bool) {
	if idMap, exist := db.outputs[absPath]; exist {
		if include, exist := idMap[id]; exist {
			return include, true
		}
	}

	return nil, false
}

// inputOutputIncludeNotExist returns an error when an input or output include
// with the given abspath and id exist.
func (db *IncludeDB) inputOutputIncludeNotExist(absPath, id string) error {
	if _, exist := db.inputInclude(absPath, id); exist {
		return fmt.Errorf("input include %q already exist, include specifiers must be unique", includeSpecifier(absPath, id))
	}

	if _, exist := db.outputInclude(absPath, id); exist {
		return fmt.Errorf("output include %q already exist, include specifiers must be unique", includeSpecifier(absPath, id))
	}

	return nil
}

func includeSpecifier(absPath, id string) string {
	return absPath + includeIDSep + id
}
