package comment

import (
	"go/ast"
	"go/token"
	"strings"
	"unicode"
)

type Comment struct {
	ast.Comment
	StartOffset int
}

type Iter struct {
	Comments []*ast.Comment

	index         int
	commentOffset int
}

func (iter *Iter) Next() *Comment {
	if iter.index == len(iter.Comments) {
		return nil
	}

	comment := iter.Comments[iter.index]

	var (
		slash       token.Pos
		text        string
		startOffset int
	)

	if comment.Text[1] == '*' {
		slash = comment.Slash + token.Pos(iter.commentOffset)

		remainingCommentTextStartIndex := 0
		if iter.commentOffset != 0 {
			remainingCommentTextStartIndex = iter.commentOffset
		}

		textStartIndex := 0
		if iter.commentOffset == 0 {
			textStartIndex = 2
		}

		remainingCommentText := comment.Text[remainingCommentTextStartIndex:]
		newLineIndex := strings.Index(remainingCommentText, "\n")

		if newLineIndex == -1 {
			text = remainingCommentText[textStartIndex : len(remainingCommentText)-2]
		} else {
			text = remainingCommentText[textStartIndex:newLineIndex]
			iter.commentOffset += newLineIndex + 1
		}

		startOffset = textStartIndex

		if newLineIndex <= 0 {
			iter.commentOffset = 0
			iter.index += 1
		}

	} else {
		slash = comment.Slash
		text = comment.Text[2:]
		startOffset = 2

		iter.index += 1
	}

	text = strings.TrimRightFunc(text, unicode.IsSpace)

	if text == "" {
		return iter.Next()
	}

	return &Comment{
		Comment: ast.Comment{
			Slash: slash,
			Text:  text,
		},
		StartOffset: startOffset,
	}
}
