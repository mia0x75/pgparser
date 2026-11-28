package json

import (
	"bytes"
	"cmp"
	"math"
	"sort"
	"strconv"
	"unicode/utf16"

	"github.com/mia0x75/pgparser/errors"
	"github.com/mia0x75/pgparser/hack"
	"github.com/mia0x75/pgparser/postgres"
)

// Type returns type of BinaryJSON as string.
func (bj BinaryJSON) Type() string {
	switch bj.TypeCode {
	case TypeCodeObject:
		return "OBJECT"
	case TypeCodeArray:
		return "ARRAY"
	case TypeCodeLiteral:
		if len(bj.Value) > 0 && bj.Value[0] == LiteralNil {
			return "NULL"
		}
		return "BOOLEAN"
	case TypeCodeInt64:
		return "INTEGER"
	case TypeCodeUint64:
		return "UNSIGNED INTEGER"
	case TypeCodeFloat64:
		return "DOUBLE"
	case TypeCodeString:
		return "STRING"
	default:
		return "UNKNOWN"
	}
}

// Quote is for JSON_QUOTE
func (bj BinaryJSON) Quote() string {
	str := hack.String(bj.GetString())
	return strconv.Quote(string(str))
}

// Unquote is for JSON_UNQUOTE.
func (bj BinaryJSON) Unquote() (string, error) {
	if err := bj.validate(0); err != nil {
		return "", err
	}
	switch bj.TypeCode {
	case TypeCodeString:
		// JSON parsing has already decoded escapes; a second pass corrupts literal backslashes and quotes.
		return string(bj.GetString()), nil
	default:
		out, err := bj.MarshalJSON()
		return string(out), err
	}
}

func unquoteString(s string) (string, error) {
	ret := new(bytes.Buffer)
	for i := 0; i < len(s); i++ {
		if s[i] == '\\' {
			i++
			if i == len(s) {
				return "", errors.New("Missing a closing quotation mark in string")
			}
			switch s[i] {
			case '"':
				ret.WriteByte('"')
			case 'b':
				ret.WriteByte('\b')
			case 'f':
				ret.WriteByte('\f')
			case 'n':
				ret.WriteByte('\n')
			case 'r':
				ret.WriteByte('\r')
			case 't':
				ret.WriteByte('\t')
			case '\\':
				ret.WriteByte('\\')
			case 'u':
				if len(s)-i < 5 {
					return "", errors.Errorf("Invalid unicode: %s", s[i+1:])
				}
				unit, err := strconv.ParseUint(s[i+1:i+5], 16, 16)
				if err != nil {
					return "", errors.Trace(err)
				}
				i += 4
				r := rune(unit)
				if utf16.IsSurrogate(r) {
					if r > 0xDBFF || len(s)-i < 7 || s[i+1:i+3] != `\u` {
						return "", errors.New("Invalid unicode surrogate pair")
					}
					low, err := strconv.ParseUint(s[i+3:i+7], 16, 16)
					if err != nil || low < 0xDC00 || low > 0xDFFF {
						return "", errors.New("Invalid unicode surrogate pair")
					}
					r = utf16.DecodeRune(r, rune(low))
					i += 6
				}
				ret.WriteRune(r)
			default:
				// For all other escape sequences, backslash is ignored.
				ret.WriteByte(s[i])
			}
		} else {
			ret.WriteByte(s[i])
		}
	}
	return ret.String(), nil
}

// Extract receives several path expressions as arguments, matches them in bj, and returns:
//
//	ret: target JSON matched any path expressions. maybe autowrapped as an array.
//	found: true if any path expressions matched.
func (bj BinaryJSON) Extract(pathExprList []PathExpression) (ret BinaryJSON, found bool) {
	buf := make([]BinaryJSON, 0, 1)
	for _, pathExpr := range pathExprList {
		buf = bj.extractTo(buf, pathExpr)
	}
	switch {
	case len(buf) == 0:
		found = false
	case len(pathExprList) == 1 && len(buf) == 1:
		// If pathExpr contains asterisks, len(elemList) won't be 1
		// even if len(pathExprList) equals to 1.
		found = true
		ret = buf[0]
	default:
		found = true
		ret = buildBinaryArray(buf)
	}
	return
}

func (bj BinaryJSON) extractTo(buf []BinaryJSON, pathExpr PathExpression) []BinaryJSON {
	if len(pathExpr.legs) == 0 {
		return append(buf, bj)
	}
	currentLeg, subPathExpr := pathExpr.popOneLeg()
	switch {
	case currentLeg.typ == pathLegIndex:
		if bj.TypeCode != TypeCodeArray {
			if currentLeg.arrayIndex <= 0 && currentLeg.arrayIndex != arrayIndexAsterisk {
				buf = bj.extractTo(buf, subPathExpr)
			}
			return buf
		}
		elemCount := bj.GetElemCount()
		if currentLeg.arrayIndex == arrayIndexAsterisk {
			for i := range elemCount {
				buf = bj.arrayGetElem(i).extractTo(buf, subPathExpr)
			}
		} else if currentLeg.arrayIndex < elemCount {
			buf = bj.arrayGetElem(currentLeg.arrayIndex).extractTo(buf, subPathExpr)
		}
	case (currentLeg.typ == pathLegKey || currentLeg.typ == pathLegKeyAsterisk) && bj.TypeCode == TypeCodeObject:
		elemCount := bj.GetElemCount()
		if currentLeg.typ == pathLegKeyAsterisk {
			for i := range elemCount {
				buf = bj.objectGetValWithCount(i, elemCount).extractTo(buf, subPathExpr)
			}
		} else {
			child, ok := bj.objectSearchKey(hack.Slice(currentLeg.dotKey))
			if ok {
				buf = child.extractTo(buf, subPathExpr)
			}
		}
	case currentLeg.typ == pathLegDoubleAsterisk:
		buf = bj.extractTo(buf, subPathExpr)
		switch bj.TypeCode {
		case TypeCodeArray:
			elemCount := bj.GetElemCount()
			for i := range elemCount {
				buf = bj.arrayGetElem(i).extractTo(buf, pathExpr)
			}
		case TypeCodeObject:
			elemCount := bj.GetElemCount()
			for i := range elemCount {
				buf = bj.objectGetValWithCount(i, elemCount).extractTo(buf, pathExpr)
			}
		}
	}
	return buf
}

func (bj BinaryJSON) objectSearchKey(key []byte) (BinaryJSON, bool) {
	elemCount := bj.GetElemCount()
	idx := sort.Search(elemCount, func(i int) bool {
		return bytes.Compare(bj.objectGetKeyWithCount(i, elemCount), key) >= 0
	})
	if idx < elemCount && bytes.Equal(bj.objectGetKeyWithCount(idx, elemCount), key) {
		return bj.objectGetValWithCount(idx, elemCount), true
	}
	return BinaryJSON{}, false
}

func buildBinaryArray(elems []BinaryJSON) BinaryJSON {
	totalSize := headerSize + len(elems)*valEntrySize
	for _, elem := range elems {
		if elem.TypeCode != TypeCodeLiteral {
			totalSize += len(elem.Value)
		}
	}
	buf := make([]byte, headerSize+len(elems)*valEntrySize, totalSize)
	endian.PutUint32(buf, uint32(len(elems)))
	endian.PutUint32(buf[dataSizeOff:], uint32(totalSize))
	buf = buildBinaryElements(buf, headerSize, elems)
	return BinaryJSON{TypeCode: TypeCodeArray, Value: buf}
}

func buildBinaryElements(buf []byte, entryStart int, elems []BinaryJSON) []byte {
	for i, elem := range elems {
		buf[entryStart+i*valEntrySize] = elem.TypeCode
		if elem.TypeCode == TypeCodeLiteral {
			buf[entryStart+i*valEntrySize+valTypeSize] = elem.Value[0]
		} else {
			endian.PutUint32(buf[entryStart+i*valEntrySize+valTypeSize:], uint32(len(buf)))
			buf = append(buf, elem.Value...)
		}
	}
	return buf
}

func buildBinaryObject(keys [][]byte, elems []BinaryJSON) BinaryJSON {
	totalSize := headerSize + len(elems)*(keyEntrySize+valEntrySize)
	for i, elem := range elems {
		if elem.TypeCode != TypeCodeLiteral {
			totalSize += len(elem.Value)
		}
		totalSize += len(keys[i])
	}
	buf := make([]byte, headerSize+len(elems)*(keyEntrySize+valEntrySize), totalSize)
	endian.PutUint32(buf, uint32(len(elems)))
	endian.PutUint32(buf[dataSizeOff:], uint32(totalSize))
	for i, key := range keys {
		endian.PutUint32(buf[headerSize+i*keyEntrySize:], uint32(len(buf)))
		endian.PutUint16(buf[headerSize+i*keyEntrySize+keyLenOff:], uint16(len(key)))
		buf = append(buf, key...)
	}
	entryStart := headerSize + len(elems)*keyEntrySize
	buf = buildBinaryElements(buf, entryStart, elems)
	return BinaryJSON{TypeCode: TypeCodeObject, Value: buf}
}

// Modify modifies a JSON object by insert, replace or set.
// All path expressions cannot contain * or ** wildcard.
// If any error occurs, the input won't be changed.
func (bj BinaryJSON) Modify(pathExprList []PathExpression, values []BinaryJSON, mt ModifyType) (retj BinaryJSON, err error) {
	if err := bj.validate(0); err != nil {
		return bj, err
	}
	for _, value := range values {
		if err := value.validate(0); err != nil {
			return bj, err
		}
	}
	if len(pathExprList) != len(values) {
		return retj, postgres.ClassJSON.New(
			postgres.ErrWrongParamcountToNativeFct,
			postgres.PgErrName[postgres.ErrWrongParamcountToNativeFct],
		).GenWithStackByArgs("JSON_MODIFY")
	}
	for _, pathExpr := range pathExprList {
		for _, leg := range pathExpr.legs {
			if leg.typ == pathLegKey && len(leg.dotKey) > math.MaxUint16 {
				return bj, errors.New("JSON object key is too long")
			}
		}
		if pathExpr.flags.containsAnyAsterisk() {
			return retj, ErrInvalidJSONPathWildcard.GenWithStackByArgs()
		}
	}
	for i := range pathExprList {
		pathExpr, value := pathExprList[i], values[i]
		modifier := &binaryModifier{bj: bj}
		switch mt {
		case ModifyInsert:
			bj = modifier.insert(pathExpr, value)
		case ModifyReplace:
			bj = modifier.replace(pathExpr, value)
		case ModifySet:
			bj = modifier.set(pathExpr, value)
		}
	}
	return bj, nil
}

// Remove removes the elements indicated by pathExprList from JSON.
func (bj BinaryJSON) Remove(pathExprList []PathExpression) (BinaryJSON, error) {
	if err := bj.validate(0); err != nil {
		return bj, err
	}
	for _, pathExpr := range pathExprList {
		if len(pathExpr.legs) == 0 {
			return bj, ErrInvalidJSONPathEmpty.GenWithStackByArgs()
		}
		if pathExpr.flags.containsAnyAsterisk() {
			return bj, ErrInvalidJSONPathWildcard.GenWithStackByArgs()
		}
		modifer := &binaryModifier{bj: bj}
		bj = modifer.remove(pathExpr)
	}
	return bj, nil
}

type binaryModifier struct {
	bj          BinaryJSON
	modifyPtr   *byte
	modifyValue BinaryJSON
}

func (bm *binaryModifier) set(path PathExpression, newBj BinaryJSON) BinaryJSON {
	result := make([]BinaryJSON, 0, 1)
	result = bm.bj.extractTo(result, path)
	if len(result) > 0 {
		bm.modifyPtr = &result[0].Value[0]
		bm.modifyValue = newBj
		return bm.rebuild()
	}
	bm.doInsert(path, newBj)
	return bm.rebuild()
}

func (bm *binaryModifier) replace(path PathExpression, newBj BinaryJSON) BinaryJSON {
	result := make([]BinaryJSON, 0, 1)
	result = bm.bj.extractTo(result, path)
	if len(result) == 0 {
		return bm.bj
	}
	bm.modifyPtr = &result[0].Value[0]
	bm.modifyValue = newBj
	return bm.rebuild()
}

func (bm *binaryModifier) insert(path PathExpression, newBj BinaryJSON) BinaryJSON {
	result := make([]BinaryJSON, 0, 1)
	result = bm.bj.extractTo(result, path)
	if len(result) > 0 {
		return bm.bj
	}
	bm.doInsert(path, newBj)
	return bm.rebuild()
}

// doInsert inserts the newBj to its parent, and builds the new parent.
func (bm *binaryModifier) doInsert(path PathExpression, newBj BinaryJSON) {
	parentPath, lastLeg := path.popOneLastLeg()
	result := make([]BinaryJSON, 0, 1)
	result = bm.bj.extractTo(result, parentPath)
	if len(result) == 0 {
		return
	}
	parentBj := result[0]
	if lastLeg.typ == pathLegIndex {
		bm.modifyPtr = &parentBj.Value[0]
		if parentBj.TypeCode != TypeCodeArray {
			bm.modifyValue = buildBinaryArray([]BinaryJSON{parentBj, newBj})
			return
		}
		elemCount := parentBj.GetElemCount()
		elems := make([]BinaryJSON, 0, elemCount+1)
		for i := range elemCount {
			elems = append(elems, parentBj.arrayGetElem(i))
		}
		elems = append(elems, newBj)
		bm.modifyValue = buildBinaryArray(elems)
		return
	}
	if parentBj.TypeCode != TypeCodeObject {
		return
	}
	bm.modifyPtr = &parentBj.Value[0]
	elemCount := parentBj.GetElemCount()
	insertKey := hack.Slice(lastLeg.dotKey)
	insertIdx := sort.Search(elemCount, func(i int) bool {
		return bytes.Compare(parentBj.objectGetKeyWithCount(i, elemCount), insertKey) >= 0
	})
	keys := make([][]byte, 0, elemCount+1)
	elems := make([]BinaryJSON, 0, elemCount+1)
	for i := range elemCount {
		if i == insertIdx {
			keys = append(keys, insertKey)
			elems = append(elems, newBj)
		}
		keys = append(keys, parentBj.objectGetKeyWithCount(i, elemCount))
		elems = append(elems, parentBj.objectGetValWithCount(i, elemCount))
	}
	if insertIdx == elemCount {
		keys = append(keys, insertKey)
		elems = append(elems, newBj)
	}
	bm.modifyValue = buildBinaryObject(keys, elems)
}

func (bm *binaryModifier) remove(path PathExpression) BinaryJSON {
	result := make([]BinaryJSON, 0, 1)
	result = bm.bj.extractTo(result, path)
	if len(result) == 0 {
		return bm.bj
	}
	bm.doRemove(path)
	return bm.rebuild()
}

func (bm *binaryModifier) doRemove(path PathExpression) {
	parentPath, lastLeg := path.popOneLastLeg()
	result := make([]BinaryJSON, 0, 1)
	result = bm.bj.extractTo(result, parentPath)
	if len(result) == 0 {
		return
	}
	parentBj := result[0]
	if lastLeg.typ == pathLegIndex {
		if parentBj.TypeCode != TypeCodeArray {
			return
		}
		bm.modifyPtr = &parentBj.Value[0]
		elemCount := parentBj.GetElemCount()
		elems := make([]BinaryJSON, 0, elemCount-1)
		for i := range elemCount {
			if i != lastLeg.arrayIndex {
				elems = append(elems, parentBj.arrayGetElem(i))
			}
		}
		bm.modifyValue = buildBinaryArray(elems)
		return
	}
	if parentBj.TypeCode != TypeCodeObject {
		return
	}
	bm.modifyPtr = &parentBj.Value[0]
	elemCount := parentBj.GetElemCount()
	removeKey := hack.Slice(lastLeg.dotKey)
	keys := make([][]byte, 0, elemCount+1)
	elems := make([]BinaryJSON, 0, elemCount+1)
	for i := range elemCount {
		key := parentBj.objectGetKeyWithCount(i, elemCount)
		if !bytes.Equal(key, removeKey) {
			keys = append(keys, key)
			elems = append(elems, parentBj.objectGetValWithCount(i, elemCount))
		}
	}
	bm.modifyValue = buildBinaryObject(keys, elems)
}

// rebuild merges the old and the modified JSON into a new BinaryJSON
func (bm *binaryModifier) rebuild() BinaryJSON {
	buf := make([]byte, 0, len(bm.bj.Value)+len(bm.modifyValue.Value))
	value, tpCode := bm.rebuildTo(buf)
	return BinaryJSON{TypeCode: tpCode, Value: value}
}

func (bm *binaryModifier) rebuildTo(buf []byte) ([]byte, TypeCode) {
	if bm.modifyPtr == &bm.bj.Value[0] {
		bm.modifyPtr = nil
		return append(buf, bm.modifyValue.Value...), bm.modifyValue.TypeCode
	} else if bm.modifyPtr == nil {
		return append(buf, bm.bj.Value...), bm.bj.TypeCode
	}
	bj := bm.bj
	switch bj.TypeCode {
	case TypeCodeLiteral, TypeCodeInt64, TypeCodeUint64, TypeCodeFloat64, TypeCodeString:
		return append(buf, bj.Value...), bj.TypeCode
	}
	docOff := len(buf)
	elemCount := bj.GetElemCount()
	var valEntryStart int
	if bj.TypeCode == TypeCodeArray {
		copySize := headerSize + elemCount*valEntrySize
		valEntryStart = headerSize
		buf = append(buf, bj.Value[:copySize]...)
	} else {
		copySize := headerSize + elemCount*(keyEntrySize+valEntrySize)
		valEntryStart = headerSize + elemCount*keyEntrySize
		buf = append(buf, bj.Value[:copySize]...)
		if elemCount > 0 {
			firstKeyOff := int(endian.Uint32(bj.Value[headerSize:]))
			lastKeyOff := int(endian.Uint32(bj.Value[headerSize+(elemCount-1)*keyEntrySize:]))
			lastKeyLen := int(endian.Uint16(bj.Value[headerSize+(elemCount-1)*keyEntrySize+keyLenOff:]))
			buf = append(buf, bj.Value[firstKeyOff:lastKeyOff+lastKeyLen]...)
		}
	}
	for i := range elemCount {
		valEntryOff := valEntryStart + i*valEntrySize
		elem := bj.valEntryGet(valEntryOff)
		bm.bj = elem
		var tpCode TypeCode
		valOff := len(buf) - docOff
		buf, tpCode = bm.rebuildTo(buf)
		buf[docOff+valEntryOff] = tpCode
		if tpCode == TypeCodeLiteral {
			lastIdx := len(buf) - 1
			endian.PutUint32(buf[docOff+valEntryOff+valTypeSize:], uint32(buf[lastIdx]))
			buf = buf[:lastIdx]
		} else {
			endian.PutUint32(buf[docOff+valEntryOff+valTypeSize:], uint32(valOff))
		}
	}
	endian.PutUint32(buf[docOff+dataSizeOff:], uint32(len(buf)-docOff))
	return buf, bj.TypeCode
}

// CompareBinary compares two binary json objects. Returns -1 if left < right,
// 0 if left == right, else returns 1.
func CompareBinary(left, right BinaryJSON) int {
	precedence1 := jsonTypePrecedences[left.Type()]
	precedence2 := jsonTypePrecedences[right.Type()]
	var cmp int
	if precedence1 == precedence2 {
		if precedence1 == jsonTypePrecedences["NULL"] {
			// for JSON null.
			cmp = 0
		}
		switch left.TypeCode {
		case TypeCodeLiteral:
			// false is less than true.
			cmp = int(right.Value[0]) - int(left.Value[0])
		case TypeCodeInt64, TypeCodeUint64, TypeCodeFloat64:
			cmp = compareBinaryNumber(left, right)
		case TypeCodeString:
			cmp = bytes.Compare(left.GetString(), right.GetString())
		case TypeCodeArray:
			leftCount := left.GetElemCount()
			rightCount := right.GetElemCount()
			for i := 0; i < leftCount && i < rightCount; i++ {
				elem1 := left.arrayGetElem(i)
				elem2 := right.arrayGetElem(i)
				cmp = CompareBinary(elem1, elem2)
				if cmp != 0 {
					return cmp
				}
			}
			cmp = leftCount - rightCount
		case TypeCodeObject:
			// only equal is defined on two json objects.
			// larger and smaller are not defined.
			cmp = bytes.Compare(left.Value, right.Value)
		}
	} else {
		cmp = precedence1 - precedence2
	}
	return cmp
}

func compareBinaryNumber(left, right BinaryJSON) int {
	if left.TypeCode == TypeCodeFloat64 {
		if right.TypeCode == TypeCodeFloat64 {
			return cmp.Compare(left.GetFloat64(), right.GetFloat64())
		}
		return -compareIntegerFloat(right, left.GetFloat64())
	}
	if right.TypeCode == TypeCodeFloat64 {
		return compareIntegerFloat(left, right.GetFloat64())
	}
	if left.TypeCode == TypeCodeInt64 && right.TypeCode == TypeCodeInt64 {
		return cmp.Compare(left.GetInt64(), right.GetInt64())
	}
	if left.TypeCode == TypeCodeInt64 && left.GetInt64() < 0 {
		return -1
	}
	if right.TypeCode == TypeCodeInt64 && right.GetInt64() < 0 {
		return 1
	}
	return cmp.Compare(left.GetUint64(), right.GetUint64())
}

func compareIntegerFloat(integer BinaryJSON, f float64) int {
	if math.IsNaN(f) {
		return 1
	}
	var result int
	// Convert the float only inside the integer range, keeping all integer bits for comparison.
	if integer.TypeCode == TypeCodeInt64 {
		if f < math.MinInt64 {
			return 1
		}
		if f >= -float64(math.MinInt64) {
			return -1
		}
		result = cmp.Compare(integer.GetInt64(), int64(f))
	} else {
		if f < 0 {
			return 1
		}
		if f >= 0x1p64 {
			return -1
		}
		result = cmp.Compare(integer.GetUint64(), uint64(f))
	}
	if result != 0 {
		return result
	}
	return cmp.Compare(math.Trunc(f), f)
}

// MergeBinary merges multiple BinaryJSON into one according the following rules:
// 1) adjacent arrays are merged to a single array;
// 2) adjacent object are merged to a single object;
// 3) a scalar value is autowrapped as an array before merge;
// 4) an adjacent array and object are merged by autowrapping the object as an array.
func MergeBinary(bjs []BinaryJSON) BinaryJSON {
	var remain = bjs
	var objects []BinaryJSON
	var results []BinaryJSON
	for len(remain) > 0 {
		if remain[0].TypeCode != TypeCodeObject {
			results = append(results, remain[0])
			remain = remain[1:]
		} else {
			objects, remain = getAdjacentObjects(remain)
			results = append(results, mergeBinaryObject(objects))
		}
	}
	if len(results) == 1 {
		return results[0]
	}
	return mergeBinaryArray(results)
}

func getAdjacentObjects(bjs []BinaryJSON) (objects, remain []BinaryJSON) {
	for i := range bjs {
		if bjs[i].TypeCode != TypeCodeObject {
			return bjs[:i], bjs[i:]
		}
	}
	return bjs, nil
}

func mergeBinaryArray(elems []BinaryJSON) BinaryJSON {
	buf := make([]BinaryJSON, 0, len(elems))
	for i := range elems {
		elem := elems[i]
		if elem.TypeCode != TypeCodeArray {
			buf = append(buf, elem)
		} else {
			childCount := elem.GetElemCount()
			for j := range childCount {
				buf = append(buf, elem.arrayGetElem(j))
			}
		}
	}
	return buildBinaryArray(buf)
}

func mergeBinaryObject(objects []BinaryJSON) BinaryJSON {
	keyValMap := make(map[string]BinaryJSON)
	keys := make([][]byte, 0, len(keyValMap))
	for _, obj := range objects {
		elemCount := obj.GetElemCount()
		for i := range elemCount {
			key := obj.objectGetKeyWithCount(i, elemCount)
			val := obj.objectGetValWithCount(i, elemCount)
			if old, ok := keyValMap[string(key)]; ok {
				keyValMap[string(key)] = MergeBinary([]BinaryJSON{old, val})
			} else {
				keyValMap[string(key)] = val
				keys = append(keys, key)
			}
		}
	}
	sort.Slice(keys, func(i, j int) bool {
		return bytes.Compare(keys[i], keys[j]) < 0
	})
	values := make([]BinaryJSON, len(keys))
	for i, key := range keys {
		values[i] = keyValMap[string(key)]
	}
	return buildBinaryObject(keys, values)
}

// PeekBytesAsJSON trys to peek some bytes from b, until
// we can deserialize a JSON from those bytes.
func PeekBytesAsJSON(b []byte) (n int, err error) {
	if len(b) == 0 {
		return 0, errors.New("cannot peek from empty bytes")
	}
	size, err := binaryValueSize(b[0], b[valTypeSize:])
	if err != nil {
		return 0, err
	}
	n = valTypeSize + size
	if err := (BinaryJSON{TypeCode: b[0], Value: b[valTypeSize:n]}).validate(0); err != nil {
		return 0, err
	}
	return n, nil
}

// ContainsBinary check whether JSON document contains specific target according the following rules:
// 1) object contains a target object if and only if every key is contained in source object and the value associated with the target key is contained in the value associated with the source key;
// 2) array contains a target nonarray if and only if the target is contained in some element of the array;
// 3) array contains a target array if and only if every element is contained in some element of the array;
// 4) scalar contains a target scalar if and only if they are comparable and are equal;
func ContainsBinary(obj, target BinaryJSON) bool {
	switch obj.TypeCode {
	case TypeCodeObject:
		if target.TypeCode == TypeCodeObject {
			len := target.GetElemCount()
			for i := range len {
				key := target.objectGetKeyWithCount(i, len)
				val := target.objectGetValWithCount(i, len)
				if exp, exists := obj.objectSearchKey(key); !exists || !ContainsBinary(exp, val) {
					return false
				}
			}
			return true
		}
		return false
	case TypeCodeArray:
		if target.TypeCode == TypeCodeArray {
			len := target.GetElemCount()
			for i := range len {
				if !ContainsBinary(obj, target.arrayGetElem(i)) {
					return false
				}
			}
			return true
		}
		len := obj.GetElemCount()
		for i := range len {
			if ContainsBinary(obj.arrayGetElem(i), target) {
				return true
			}
		}
		return false
	default:
		return CompareBinary(obj, target) == 0
	}
}

// GetElemDepth for JSON_DEPTH
// Returns the maximum depth of a JSON document
// 1) An empty array, empty object, or scalar value has depth 1.
// 2) A nonempty array containing only elements of depth 1 or nonempty object containing only member values of depth 1 has depth 2.
// 3) Otherwise, a JSON document has depth greater than 2.
// e.g. depth of '{}', '[]', 'true': 1
// e.g. depth of '[10, 20]', '[[], {}]': 2
// e.g. depth of '[10, {"a": 20}]': 3
func (bj BinaryJSON) GetElemDepth() int {
	switch bj.TypeCode {
	case TypeCodeObject:
		elemCount := bj.GetElemCount()
		maxDepth := 0
		for i := range elemCount {
			obj := bj.objectGetValWithCount(i, elemCount)
			depth := obj.GetElemDepth()
			if depth > maxDepth {
				maxDepth = depth
			}
		}
		return maxDepth + 1
	case TypeCodeArray:
		elemCount := bj.GetElemCount()
		maxDepth := 0
		for i := range elemCount {
			obj := bj.arrayGetElem(i)
			depth := obj.GetElemDepth()
			if depth > maxDepth {
				maxDepth = depth
			}
		}
		return maxDepth + 1
	default:
		return 1
	}
}
