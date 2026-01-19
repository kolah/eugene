package loader

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pb33f/libopenapi"
	"github.com/pb33f/libopenapi/datamodel"
	v3 "github.com/pb33f/libopenapi/datamodel/high/v3"
)

type Result struct {
	Document *libopenapi.DocumentModel[v3.Document]
	Version  string
	Warnings []string
	RawData  []byte
}

func LoadFile(path string) (*Result, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading spec file: %w", err)
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("resolving absolute path: %w", err)
	}

	config := &datamodel.DocumentConfiguration{
		BasePath:            filepath.Dir(absPath),
		AllowFileReferences: true,
	}

	return loadWithConfig(data, config)
}

func loadWithConfig(data []byte, config *datamodel.DocumentConfiguration) (*Result, error) {
	var doc libopenapi.Document
	var err error

	if config != nil {
		doc, err = libopenapi.NewDocumentWithConfiguration(data, config)
	} else {
		doc, err = libopenapi.NewDocument(data)
	}
	if err != nil {
		return nil, fmt.Errorf("parsing OpenAPI document: %w", err)
	}

	version := doc.GetVersion()
	if !strings.HasPrefix(version, "3.") {
		return nil, fmt.Errorf("unsupported OpenAPI version: %s (only 3.x supported)", version)
	}

	model, err := doc.BuildV3Model()
	if err != nil {
		return nil, fmt.Errorf("building OpenAPI model: %w", err)
	}

	result := &Result{
		Document: model,
		Version:  version,
		RawData:  data,
	}

	if strings.HasPrefix(version, "3.0") {
		result.Warnings = append(result.Warnings, "OpenAPI 3.0.x detected; some 3.1/3.2 features unavailable")
	}

	return result, nil
}
