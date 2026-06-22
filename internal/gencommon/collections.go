package gencommon

import "strings"

// ParseCQLInner extracts the inner type(s) from CQL collection wrappers.
// "set<uuid>" → ("uuid", ""), "list<text>" → ("text", ""), "map<text,int>" → ("text","int")
func ParseCQLInner(sl string) (string, string) {
	for _, prefix := range []string{"set<", "list<", "frozen<"} {
		if strings.HasPrefix(sl, prefix) && strings.HasSuffix(sl, ">") {
			return strings.TrimSpace(sl[len(prefix) : len(sl)-1]), ""
		}
	}
	if strings.HasPrefix(sl, "map<") && strings.HasSuffix(sl, ">") {
		inner := sl[4 : len(sl)-1]
		depth := 0
		for i, ch := range inner {
			switch ch {
			case '<':
				depth++
			case '>':
				depth--
			case ',':
				if depth == 0 {
					return strings.TrimSpace(inner[:i]), strings.TrimSpace(inner[i+1:])
				}
			}
		}
	}
	return "text", ""
}

// ExtractCQLInner extracts the inner type from a CQL collection wrapper like frozen<X> or list<X>.
func ExtractCQLInner(typ, wrapper string) (string, bool) {
	prefix := wrapper + "<"
	if strings.HasPrefix(typ, prefix) && strings.HasSuffix(typ, ">") {
		inner := typ[len(prefix) : len(typ)-1]
		return strings.TrimSpace(inner), true
	}
	return "", false
}

// ExtractCollectionInner extracts the element type from set<T>, list<T>, or frozen<T>.
func ExtractCollectionInner(typ string) string {
	for _, wrapper := range []string{"set", "list", "frozen"} {
		if inner, ok := ExtractCQLInner(typ, wrapper); ok {
			return inner
		}
	}
	return "text"
}

// ExtractMapTypes extracts key and value types from map<K,V>.
func ExtractMapTypes(typ string) (string, string) {
	k, v, ok := ExtractCQLMap(typ)
	if !ok {
		return "string", "string"
	}
	return k, v
}

// ExtractCQLMap extracts key and value types from a CQL map<key,value>.
func ExtractCQLMap(typ string) (string, string, bool) {
	if !strings.HasPrefix(typ, "map<") || !strings.HasSuffix(typ, ">") {
		return "", "", false
	}
	inner := typ[4 : len(typ)-1]
	angle := 0
	for i, ch := range inner {
		switch ch {
		case '<':
			angle++
		case '>':
			angle--
		case ',':
			if angle == 0 {
				key := strings.TrimSpace(inner[:i])
				val := strings.TrimSpace(inner[i+1:])
				return key, val, true
			}
		}
	}
	return "", "", false
}
