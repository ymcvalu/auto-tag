package main

import (
	"bytes"
	"errors"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"log"
	"os"
	"strconv"
)

func main() {
	args := os.Args[:len(os.Args)]
	if len(args) < 4 {
		log.Fatal("参数：文件路径，行号，列号")
	}
	fpath := args[1]
	lineNum, err := strconv.Atoi(args[2])
	if err != nil {
		log.Fatal("incorrect line number")
	}
	// columnNum, err := strconv.Atoi(args[3])
	if err != nil {
		log.Fatal("incorrect column number")
	}

	fset := token.NewFileSet()

	// 解析注释
	f, err := parser.ParseFile(fset, fpath, nil, parser.ParseComments)
	if err != nil {
		log.Fatal("failed to parse file: ", err.Error())
	}

	var target *ast.StructType
	ast.Inspect(f, func(node ast.Node) bool {
		// 如果不是结构体类型声明，跳过，继续下一个遍历
		st, ok := node.(*ast.StructType)
		if !ok || st.Incomplete {
			return true
		}

		begin := fset.Position(st.Pos())
		end := fset.Position(st.End())

		// 找到目标struct
		if begin.Line <= lineNum && end.Line >= lineNum {
			target = st
			return false
		}

		return true
	})

	if target != nil {
		genTag(target)
	}

	fd, err := os.OpenFile(fpath, os.O_TRUNC|os.O_RDWR, 0777)

	if err != nil {
		log.Fatal(err)
	}
	defer fd.Close()
	err = format.Node(fd, fset, f)
	if err != nil {
		log.Fatal(err)
	}
}

func genTag(st *ast.StructType) {
	fs := st.Fields.List
	for i := range fs {
		var (
			tag string
		)

		fd := fs[i]
		if len(fd.Names) > 0 {
			name := fd.Names[0].Name
			if !isExport(name) {
				continue
			}
			tag = genKey(name)
		}

		switch t := fd.Type.(type) {
		case *ast.Ident:
			if tag == "" && isExport(t.Name) {
				tag = genKey(t.Name)
			}
		case *ast.StructType:
			genTag(t)
		}

		var tagStr string
		if fd.Tag != nil {
			tagStr = fd.Tag.Value
		}

		tags, err := parseTag(tagStr)
		if err != nil {
			log.Fatal(err)
		}

		change := false
		if _, ok := tags.Lookup("json"); !ok {
			tags.Append("json", tag)
			change = true
		}
		if _, ok := tags.Lookup("form"); !ok {
			tags.Append("form", tag)
			change = true
		}

		if change {
			tagStr = tags.TagStr()
			if fd.Tag == nil {
				fd.Tag = &ast.BasicLit{}
			}
			fd.Tag.Kind = token.STRING
			fd.Tag.Value = tagStr
		}
	}

}

func isExport(name string) bool {
	if len(name) == 0 {
		return false
	}
	if name[0] >= 'A' && name[0] <= 'Z' {
		return true
	}
	return false
}

func genKey(name string) string {
	if name == "" {
		return ""
	}
	buf := bytes.NewBuffer(nil)
	if name[0] >= 'A' && name[0] <= 'Z' {
		buf.WriteByte(name[0] - 'A' + 'a')
	} else {
		buf.WriteByte(name[0])
	}

	preUpper := true

	for i := 1; i < len(name); i++ {
		c := name[i]
		if c >= 'A' && c <= 'Z' {
			if preUpper {
				buf.WriteByte(c - 'A' + 'a')
			} else {
				preUpper = true
				buf.WriteByte('_')
				buf.WriteByte(c - 'A' + 'a')
			}
		} else {
			buf.WriteByte(c)
			if preUpper {
				preUpper = false
			}
		}
	}

	return buf.String()
}

type Tag struct {
	Key string
	Val string
}

func parseTag(tagStr string) (tags Tags, err error) {
	if tagStr == "" {
		return
	}

	if len(tagStr) < 2 {
		err = errors.New("invalid tag string")
		return
	}

	tagStr = tagStr[1 : len(tagStr)-1]

	for tagStr != "" {
		// Skip leading space.
		i := 0
		for i < len(tagStr) && tagStr[i] == ' ' {
			i++
		}
		tagStr = tagStr[i:]
		if tagStr == "" {
			break
		}

		// Scan to colon. A space, a quote or a control character is a syntax error.
		// Strictly speaking, control chars include the range [0x7f, 0x9f], not just
		// [0x00, 0x1f], but in practice, we ignore the multi-byte control characters
		// as it is simpler to inspect the tag's bytes than the tag's runes.
		i = 0
		for i < len(tagStr) && tagStr[i] > ' ' && tagStr[i] != ':' && tagStr[i] != '"' && tagStr[i] != 0x7f {
			i++
		}

		if i+1 >= len(tagStr) || i == 0 || tagStr[i] != ':' || tagStr[i+1] != '"' {
			err = errors.New("invalid tag syntax")
			return
		}

		key := string(tagStr[:i])
		tagStr = tagStr[i+1:]

		// Scan quoted string to find value.
		i = 1
		for i < len(tagStr) && tagStr[i] != '"' {
			if tagStr[i] == '\\' {
				i++
			}
			i++
		}
		if i >= len(tagStr) {
			break
		}

		rawVal := string(tagStr[:i+1])
		tagStr = tagStr[i+1:]
		var val string
		val, err = strconv.Unquote(rawVal)
		if err != nil {
			break
		}
		tags = append(tags, Tag{
			Key: key,
			Val: val,
		})
	}
	return
}

type Tags []Tag

func (t Tags) Lookup(key string) (val string, has bool) {
	for i := range t {
		if t[i].Key == key {
			return t[i].Val, true
		}
	}
	return "", false
}

func (t *Tags) Append(key, val string) {
	*t = append(*t, Tag{
		Key: key,
		Val: val,
	})
}

func (t Tags) TagStr() string {
	buf := bytes.NewBuffer(nil)
	buf.WriteByte('`')
	for i := range t {
		it := t[i]
		if i > 0 {
			buf.WriteByte(' ')
		}
		buf.WriteString(it.Key)
		buf.WriteByte(':')
		buf.WriteByte('"')
		buf.WriteString(it.Val)
		buf.WriteByte('"')
	}
	buf.WriteByte('`')
	return buf.String()
}
