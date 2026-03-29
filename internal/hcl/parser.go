package hcl

import (
	"fmt"

	hashcl "github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
)

// ParseFile parses a single .tf file content (as string) into a TFModule.
// filePath is used for identification only.
func ParseFile(filePath, content string) (*TFModule, error) {
	module := &TFModule{Path: filePath}

	file, diags := hclsyntax.ParseConfig([]byte(content), filePath, hashcl.Pos{Line: 1, Column: 1})
	if diags.HasErrors() {
		module.Path = filePath + ":parse-error"
		return module, fmt.Errorf("parse error in %s: %s", filePath, diags.Error())
	}

	body, ok := file.Body.(*hclsyntax.Body)
	if !ok {
		return module, nil
	}

	for _, block := range body.Blocks {
		switch block.Type {
		case "terraform":
			parseTerraformBlock(block, module)
		case "resource":
			if len(block.Labels) >= 2 {
				res := parseResourceBlock(block)
				module.Resources = append(module.Resources, res)
			}
		case "module":
			if len(block.Labels) >= 1 {
				mc := parseModuleCallBlock(block)
				module.Modules = append(module.Modules, mc)
			}
		case "variable":
			if len(block.Labels) >= 1 {
				v := parseVariableBlock(block)
				module.Variables = append(module.Variables, v)
			}
		}
	}

	return module, nil
}

func parseTerraformBlock(block *hclsyntax.Block, module *TFModule) {
	for _, inner := range block.Body.Blocks {
		switch inner.Type {
		case "required_providers":
			parseRequiredProvidersBlock(inner, module)
		case "backend":
			parseBackendBlock(inner, module)
		}
	}
}

func parseRequiredProvidersBlock(block *hclsyntax.Block, module *TFModule) {
	// Each attribute is a provider.
	for name, attr := range block.Body.Attributes {
		provider := parseProviderAttribute(name, attr)
		module.Providers = append(module.Providers, provider)
	}
}

func parseProviderAttribute(name string, attr *hclsyntax.Attribute) TFProvider {
	provider := TFProvider{}
	if objExpr, ok := attr.Expr.(*hclsyntax.ObjectConsExpr); ok {
		setProviderFields(&provider, objExpr)
	}
	if provider.Source == "" {
		provider.Source = name
	}
	return provider
}

func setProviderFields(provider *TFProvider, objExpr *hclsyntax.ObjectConsExpr) {
	for _, item := range objExpr.Items {
		keyVal := consKeyToString(item.KeyExpr)
		switch keyVal {
		case "source":
			provider.Source = exprToString(item.ValueExpr)
		case "version":
			provider.Version = exprToString(item.ValueExpr)
		}
	}
}

func parseBackendBlock(block *hclsyntax.Block, module *TFModule) {
	if len(block.Labels) < 1 {
		return
	}

	backend := &TFBackend{
		Type:   block.Labels[0],
		Config: make(map[string]string),
	}
	for name, attr := range block.Body.Attributes {
		backend.Config[name] = exprToString(attr.Expr)
	}
	module.Backend = backend
}

func parseResourceBlock(block *hclsyntax.Block) TFResource {
	res := TFResource{
		Type:   block.Labels[0],
		Name:   block.Labels[1],
		Labels: make(map[string]string),
	}

	// Try to get tags or labels attribute
	for _, attrName := range []string{"tags", "labels"} {
		if attr, ok := block.Body.Attributes[attrName]; ok {
			if objExpr, ok := attr.Expr.(*hclsyntax.ObjectConsExpr); ok {
				for _, item := range objExpr.Items {
					key := consKeyToString(item.KeyExpr)
					val := exprToString(item.ValueExpr)
					res.Labels[key] = val
				}
			}
		}
	}

	return res
}

func parseModuleCallBlock(block *hclsyntax.Block) TFModuleCall {
	mc := TFModuleCall{Name: block.Labels[0]}
	if attr, ok := block.Body.Attributes["source"]; ok {
		mc.Source = exprToString(attr.Expr)
	}
	return mc
}

func parseVariableBlock(block *hclsyntax.Block) TFVariable {
	v := TFVariable{Name: block.Labels[0]}
	if attr, ok := block.Body.Attributes["type"]; ok {
		v.Type = exprToString(attr.Expr)
	}
	return v
}

// consKeyToString extracts a string from an ObjectConsKeyExpr (which wraps various expression types).
func consKeyToString(expr hclsyntax.Expression) string {
	if keyExpr, ok := expr.(*hclsyntax.ObjectConsKeyExpr); ok {
		// If ForceNonLiteral is false, the wrapped expression is a traversal used as a literal key
		return exprToString(keyExpr.Wrapped)
	}
	return exprToString(expr)
}

// exprToString tries to get a string value from an expression.
func exprToString(expr hclsyntax.Expression) string {
	switch e := expr.(type) {
	case *hclsyntax.LiteralValueExpr:
		return e.Val.AsString()
	case *hclsyntax.TemplateExpr:
		if len(e.Parts) == 1 {
			return exprToString(e.Parts[0])
		}
		// Concatenate parts
		result := ""
		for _, part := range e.Parts {
			result += exprToString(part)
		}
		return result
	case *hclsyntax.TemplateWrapExpr:
		return exprToString(e.Wrapped)
	case *hclsyntax.ScopeTraversalExpr:
		// For type references like string, number, bool
		if len(e.Traversal) > 0 {
			return e.Traversal.RootName()
		}
	}
	return ""
}
