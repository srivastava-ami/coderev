package treesitter

import (
	"slices"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/golang"
	"github.com/smacker/go-tree-sitter/javascript"
	"github.com/smacker/go-tree-sitter/python"
	"github.com/smacker/go-tree-sitter/rust"
	tstsx "github.com/smacker/go-tree-sitter/typescript/tsx"
	tsts "github.com/smacker/go-tree-sitter/typescript/typescript"

	"github.com/srivastava-ami/coderev/internal/analysis"
)

// LangDef describes the AST shape of a language so the walker can be
// language-agnostic.
type LangDef struct {
	Name           string
	GetLanguage    func() *sitter.Language
	FunctionTypes  []string // node types that open a new function scope
	BranchTypes    []string // node types that add 1 to cyclomatic complexity
	BlockTypes     []string // node types that deepen nesting
	ParameterType  string   // node type whose child count = parameter count
	ClassTypes     []string // node types that represent classes/structs
	CommentTypes   []string
}

var langDefs = map[analysis.Language]*LangDef{
	analysis.LangTypeScript: {
		Name:        "typescript",
		GetLanguage: tsts.GetLanguage,
		FunctionTypes: []string{
			"function_declaration", "function_expression",
			"arrow_function", "method_definition", "generator_function_declaration",
		},
		BranchTypes: []string{
			"if_statement", "else_clause", "while_statement", "do_statement",
			"for_statement", "for_in_statement", "switch_case",
			"catch_clause", "ternary_expression",
		},
		BlockTypes:    []string{"statement_block"},
		ParameterType: "formal_parameters",
		ClassTypes:    []string{"class_declaration", "class_expression"},
		CommentTypes:  []string{"comment"},
	},
	analysis.LangJavaScript: {
		Name:        "javascript",
		GetLanguage: javascript.GetLanguage,
		FunctionTypes: []string{
			"function_declaration", "function_expression",
			"arrow_function", "method_definition", "generator_function_declaration",
		},
		BranchTypes: []string{
			"if_statement", "else_clause", "while_statement", "do_statement",
			"for_statement", "for_in_statement", "switch_case",
			"catch_clause", "ternary_expression",
		},
		BlockTypes:    []string{"statement_block"},
		ParameterType: "formal_parameters",
		ClassTypes:    []string{"class_declaration", "class_expression"},
		CommentTypes:  []string{"comment"},
	},
	analysis.LangGo: {
		Name:        "go",
		GetLanguage: golang.GetLanguage,
		FunctionTypes: []string{
			"function_declaration", "method_declaration", "func_literal",
		},
		BranchTypes: []string{
			"if_statement", "for_statement", "switch_statement",
			"expression_case", "type_case", "select_statement", "comm_clause",
		},
		BlockTypes:    []string{"block"},
		ParameterType: "parameter_list",
		ClassTypes:    []string{"type_declaration"}, // struct types
		CommentTypes:  []string{"comment"},
	},
	analysis.LangPython: {
		Name:        "python",
		GetLanguage: python.GetLanguage,
		FunctionTypes: []string{
			"function_definition", "lambda",
		},
		BranchTypes: []string{
			"if_statement", "elif_clause", "for_statement", "while_statement",
			"except_clause", "conditional_expression", "with_statement",
			"match_statement", "case_clause",
		},
		BlockTypes:    []string{"block"},
		ParameterType: "parameters",
		ClassTypes:    []string{"class_definition"},
		CommentTypes:  []string{"comment"},
	},
	analysis.LangRust: {
		Name:        "rust",
		GetLanguage: rust.GetLanguage,
		FunctionTypes: []string{
			"function_item", "closure_expression",
		},
		BranchTypes: []string{
			"if_expression", "else_clause", "while_expression", "for_expression",
			"loop_expression", "match_arm", "if_let_expression", "while_let_expression",
		},
		BlockTypes:    []string{"block"},
		ParameterType: "parameters",
		ClassTypes:    []string{"struct_item", "impl_item"},
		CommentTypes:  []string{"line_comment", "block_comment"},
	},
}

// tsxDef is identical to TypeScript but uses the TSX grammar.
var tsxDef = &LangDef{
	Name:        "tsx",
	GetLanguage: tstsx.GetLanguage,
	FunctionTypes: []string{
		"function_declaration", "function_expression",
		"arrow_function", "method_definition", "generator_function_declaration",
	},
	BranchTypes: []string{
		"if_statement", "else_clause", "while_statement", "do_statement",
		"for_statement", "for_in_statement", "switch_case",
		"catch_clause", "ternary_expression",
	},
	BlockTypes:    []string{"statement_block"},
	ParameterType: "formal_parameters",
	ClassTypes:    []string{"class_declaration", "class_expression"},
	CommentTypes:  []string{"comment"},
}

func defFor(lang analysis.Language) *LangDef {
	if d, ok := langDefs[lang]; ok {
		return d
	}
	return nil
}

func contains(haystack []string, needle string) bool {
	return slices.Contains(haystack, needle)
}
