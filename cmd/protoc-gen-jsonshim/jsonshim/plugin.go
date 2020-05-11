// Copyright 2019 Istio Authors
//
//   Licensed under the Apache License, Version 2.0 (the "License");
//   you may not use this file except in compliance with the License.
//   You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
//   Unless required by applicable law or agreed to in writing, software
//   distributed under the License is distributed on an "AS IS" BASIS,
//   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//   See the License for the specific language governing permissions and
//   limitations under the License.

package jsonshim

import (
	"path"
	"strings"

	"github.com/gogo/protobuf/gogoproto"
	"github.com/gogo/protobuf/protoc-gen-gogo/generator"
)

func init() {
	generator.RegisterPlugin(NewPlugin())
}

// FileNameSuffix is the suffix added to files generated by jsonshim
const FileNameSuffix = "_json.gen.go"

// Plugin is a protoc-gen-gogo plugin that creates MarshalJSON() and
// UnmarshalJSON() functions for protobuf types.
type Plugin struct {
	*generator.Generator
	generator.PluginImports
	filesWritten map[string]interface{}
}

// NewPlugin returns a new instance of the Plugin
func NewPlugin() *Plugin {
	return &Plugin{
		filesWritten: map[string]interface{}{},
	}
}

// Name returns the name of this plugin
func (p *Plugin) Name() string {
	return "jsonshim"
}

// Init initializes our plugin with the active generator
func (p *Plugin) Init(g *generator.Generator) {
	p.Generator = g
}

// Generate our content
func (p *Plugin) Generate(file *generator.FileDescriptor) {
	p.PluginImports = generator.NewPluginImports(p.Generator)

	// imported packages
	bytesPkg := p.NewImport("bytes")
	jsonpbPkg := p.NewImport("github.com/gogo/protobuf/jsonpb")

	wroteMarshalers := false
	marshalerName := generator.FileName(file) + "Marshaler"
	unmarshalerName := generator.FileName(file) + "Unmarshaler"
	for _, message := range file.Messages() {
		// check to make sure something was generated for this type
		if !gogoproto.HasTypeDecl(file.FileDescriptorProto, message.DescriptorProto) {
			continue
		}

		// skip maps in protos.
		if message.Options != nil && message.Options.GetMapEntry() {
			continue
		}

		typeName := generator.CamelCaseSlice(message.TypeName())

		// Generate MarshalJSON() method for this type
		p.P(`// MarshalJSON is a custom marshaler for `, typeName)
		p.P(`func (this *`, typeName, `) MarshalJSON() ([]byte, error) {`)
		p.In()
		p.P(`str, err := `, marshalerName, `.MarshalToString(this)`)
		p.P(`return []byte(str), err`)
		p.Out()
		p.P(`}`)

		// Generate UnmarshalJSON() method for this type
		p.P(`// UnmarshalJSON is a custom unmarshaler for `, typeName)
		p.P(`func (this *`, typeName, `) UnmarshalJSON(b []byte) error {`)
		p.In()
		p.P(`return `, unmarshalerName, `.Unmarshal(`, bytesPkg.Use(), `.NewReader(b), this)`)
		p.Out()
		p.P(`}`)

		wroteMarshalers = true
	}

	if !wroteMarshalers {
		return
	}

	// write out globals
	p.P(`var (`)
	p.In()
	p.P(marshalerName, ` = &`, jsonpbPkg.Use(), `.Marshaler{}`)
	p.P(unmarshalerName, ` = &`, jsonpbPkg.Use(), `.Unmarshaler{AllowUnknownFields: true}`)
	p.Out()
	p.P(`)`)

	// store this file away
	p.addFile(file)
}

func (p *Plugin) addFile(file *generator.FileDescriptor) {
	name := file.GetName()
	importPath := ""
	// the relevant bits of FileDescriptor.goPackageOption(), if only it were exported
	opt := file.GetOptions().GetGoPackage()
	if opt != "" {
		if sc := strings.Index(opt, ";"); sc >= 0 {
			// A semicolon-delimited suffix delimits the import path and package name.
			importPath = opt[:sc]
		} else if strings.LastIndex(opt, "/") > 0 {
			// The presence of a slash implies there's an import path.
			importPath = opt
		}
	}
	// strip the extension
	name = name[:len(name)-len(path.Ext(name))]
	if importPath != "" {
		name = path.Join(importPath, path.Base(name))
	}
	p.filesWritten[name+FileNameSuffix] = struct{}{}
}

// FilesWritten returns a list of the names of files for which output was generated
func (p *Plugin) FilesWritten() map[string]interface{} {
	return p.filesWritten
}
