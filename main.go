package main

import (
	"fmt"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/hclwrite"
)

var somejunk = []byte(
	"locals {\n" +
		"common_tags = map(\n\"foo\", \"bar\", \"omg\", \"yee\",\n)\n" +
		"}\n" +
		"resource \"some\" \"thing\" {\ntags = map(\n\"hiii\", map(\"why\",((\"u\")),\"nest\")\n)\n}\n",
)

func main() {
	f, diags := hclwrite.ParseConfig(somejunk, "", hcl.Pos{Line: 1, Column: 1})
	if diags.HasErrors() {
		fmt.Printf("errors: %s", diags)
		return
	}

	for _, block := range f.Body().Blocks() {
		body := block.Body()
		attrs := body.Attributes()
		for name, attr := range attrs {
			fmt.Println("omggg: ", name, attr)
			fmt.Printf("v is %T\n", attr)
			tokens := attr.Expr().BuildTokens(nil)
			//tokens := v.BuildTokens(nil)
			fmt.Printf("token: %s", tokens[0].Bytes)
			if string(tokens[0].Bytes) == "map" {
				body.SetAttributeRaw(name, rewriteMap(tokens))
			}
		}
	}

	fmt.Printf("====\nBytes:\n\n%s\n", f.Bytes())

}

func rewriteMap(tokens hclwrite.Tokens) hclwrite.Tokens {
	mapFunc := tokens[0]
	oParen := tokens[1]
	cParen := tokens[len(tokens)-1]

	if string(mapFunc.Bytes) != "map" || oParen.Type != hclsyntax.TokenOParen || cParen.Type != hclsyntax.TokenCParen {
		return tokens
	}

	*tokens[0] = hclwrite.Token{}
	*tokens[1] = hclwrite.Token{
		Type:  hclsyntax.TokenOBrace,
		Bytes: []byte("{"),
	}
	*tokens[len(tokens)-1] = hclwrite.Token{
		Type:  hclsyntax.TokenCBrace,
		Bytes: []byte("}"),
	}

	rewriteCommas := 0

	brack := 0
	paren := 0
	brace := 0

	for i := 2; i < len(tokens); i++ {
		token := *tokens[i]

		switch token.Type {
		case hclsyntax.TokenOBrace:
			brace++
		case hclsyntax.TokenCBrace:
			brace--
		case hclsyntax.TokenOParen:
			paren++
		case hclsyntax.TokenCParen:
			paren--
		case hclsyntax.TokenOBrack:
			brack++
		case hclsyntax.TokenCBrack:
			brack--
		case hclsyntax.TokenComma:
			// If we're inside a collection, don't rewrite
			if brack > 0 || paren > 0 || brace > 0 {
				continue
			}

			if rewriteCommas%2 == 0 {
				*tokens[i] = hclwrite.Token{
					Type:         hclsyntax.TokenEqual,
					Bytes:        []byte{'='},
					SpacesBefore: token.SpacesBefore,
				}
			} else {
				*tokens[i] = hclwrite.Token{
					Type:  hclsyntax.TokenNewline,
					Bytes: []byte("\n"),
				}
			}
			rewriteCommas++

		case hclsyntax.TokenIdent:
			if string(token.Bytes) != "map" {
				continue
			}

			/*
			 * having found the start of a nested map, we need to
			 * find the index of the closing paren. The paren
			 * counter is set to 1 and the loop begins from the first
			 * token after the opening paren
			 */
			sliceEnd := i + 2
			mapParens := 1

			for sliceEnd < len(tokens) {
				fmt.Println("inner bytes: ", string(tokens[sliceEnd].Bytes))
				switch tokens[sliceEnd].Type {
				case hclsyntax.TokenOParen:
					mapParens++
				case hclsyntax.TokenCParen:
					mapParens--
				}
				sliceEnd++
				if mapParens == 0 {
					break
				}
			}

			newTokens := rewriteMap(tokens[i:sliceEnd])
			for idx, newToken := range newTokens {
				*tokens[i+idx] = *newToken
				fmt.Println("new bytes: ", string(newToken.Bytes))
			}
			//i = 1 + len(newTokens)

			fmt.Println("ident: ", string(token.Bytes))
		}
	}

	return tokens
}
