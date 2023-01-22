package loader

import (
	"fmt"
	"github.com/f1bonacc1/process-compose/src/types"
	"github.com/imdario/mergo"
	"reflect"
	"sort"
	"strings"
)

type specials struct {
	m map[reflect.Type]func(dst, src reflect.Value) error
}

var processSpecials = &specials{
	m: map[reflect.Type]func(dst, src reflect.Value) error{
		reflect.TypeOf(types.Environment{}): mergeSlice(toEnvVarMap, toEnvVarSlice),
	},
}

var projectSpecials = &specials{
	m: map[reflect.Type]func(dst, src reflect.Value) error{
		reflect.TypeOf(types.Environment{}): mergeSlice(toEnvVarMap, toEnvVarSlice),
		reflect.TypeOf(types.Processes{}):   specialProcessesMerge,
	},
}

func (s *specials) Transformer(t reflect.Type) func(dst, src reflect.Value) error {
	if fn, ok := s.m[t]; ok {
		return fn
	}
	return nil
}

type toMapFn func(s interface{}) (map[interface{}]interface{}, error)
type writeValueFromMapFn func(reflect.Value, map[interface{}]interface{}) error

func mergeSlice(toMap toMapFn, writeValue writeValueFromMapFn) func(dst, src reflect.Value) error {
	return func(dst, src reflect.Value) error {
		dstMap, err := sliceToMap(toMap, dst)
		if err != nil {
			return err
		}
		srcMap, err := sliceToMap(toMap, src)
		if err != nil {
			return err
		}
		if err := mergo.Map(&dstMap, srcMap, mergo.WithOverride); err != nil {
			return err
		}
		return writeValue(dst, dstMap)
	}
}

func sliceToMap(toMap toMapFn, v reflect.Value) (map[interface{}]interface{}, error) {
	// check if valid
	if !v.IsValid() {
		return nil, fmt.Errorf("invalid value : %+v", v)
	}
	return toMap(v.Interface())
}

func toEnvVarMap(s interface{}) (map[interface{}]interface{}, error) {
	envVars, ok := s.(types.Environment)
	if !ok {
		return nil, fmt.Errorf("not an Environment slice: %v", s)
	}
	m := map[interface{}]interface{}{}
	for _, v := range envVars {
		kv := strings.Split(v, "=")
		if len(kv) == 2 {
			m[kv[0]] = kv[1]
		}
	}
	return m, nil
}

func toEnvVarSlice(dst reflect.Value, m map[interface{}]interface{}) error {
	var s types.Environment
	for k, v := range m {
		kv := fmt.Sprintf("%s=%s", k.(string), v.(string))
		s = append(s, kv)
	}
	sort.Strings(s)
	dst.Set(reflect.ValueOf(s))
	return nil
}

func merge(opts *LoaderOptions) (*types.Project, error) {
	base := opts.projects[0]
	if len(opts.projects) == 1 {
		return base, nil
	}

	for i, override := range opts.projects[1:] {
		if err := mergeProjects(base, override); err != nil {
			return base, fmt.Errorf("cannot merge projects from %s - %v", opts.FileNames[i], err)
		}
	}
	return base, nil

}
func specialProcessesMerge(dst, src reflect.Value) error {
	if !dst.IsValid() {
		return fmt.Errorf("invalid value: %+v", dst)
	}
	if !src.IsValid() {
		return fmt.Errorf("invalid value: %+v", src)
	}
	var (
		dstProc types.Processes
		srcProc types.Processes
		ok      bool
	)
	if dstProc, ok = dst.Interface().(types.Processes); !ok {
		return fmt.Errorf("invalid type: %+v", dst)
	}
	if srcProc, ok = src.Interface().(types.Processes); !ok {
		return fmt.Errorf("invalid type: %+v", src)
	}
	merged, err := mergeProcesses(dstProc, srcProc)
	dst.Set(reflect.ValueOf(merged))
	return err
}
func mergeProcesses(base, override types.Processes) (types.Processes, error) {
	for name, overrideProcess := range override {
		overrideProcess := overrideProcess
		if baseProcess, ok := base[name]; ok {
			merged, err := mergeProcess(&baseProcess, &overrideProcess)
			if err != nil {
				return nil, fmt.Errorf("cannot merge process %s - %v", name, err)
			}
			base[name] = *merged
			continue
		}
		base[name] = overrideProcess
	}

	return base, nil
}

func mergeProcess(base, override *types.ProcessConfig) (*types.ProcessConfig, error) {
	if err := mergo.Merge(base, override,
		mergo.WithAppendSlice,
		mergo.WithOverride,
		mergo.WithTransformers(processSpecials)); err != nil {
		return nil, err
	}

	return base, nil
}

func mergeProjects(base, override *types.Project) error {
	if err := mergo.Merge(base, override,
		mergo.WithAppendSlice,
		mergo.WithOverride,
		mergo.WithTransformers(projectSpecials)); err != nil {
		return err
	}
	return nil
}
