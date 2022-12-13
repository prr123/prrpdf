// library to create pdf
// analyse pdf documents
// author: prr
// created: 2/12/2022
//
// library pdf files in go
// author: prr
// date 2/12/2022
// copyright 2022 prr azul software
//

package pdflib

import (
	"os"
	"fmt"
//	"strconv"
//	"strings"
	"bytes"
	"io"
	"compress/zlib"
	util "prrpdf/utilLib"
)

const (
	letter =iota
	landscape
	A5
	A4
	A3
	A2
	A1
)

const (
	Catalog = iota
	Pages
	Page
	Contents
	Font
	FontDescriptor
	Data
)

type InfoPdf struct {
	majver int
	minver int
	fil *os.File
	txtFil *os.File
	buf *[]byte
	filSize int64
	filNam string
//	sizeObj int
	objSize int
	numObj int
	pageCount int
	infoId int
	rootId int
	pagesId int
	xref int
	startxref int
	trailer int
	objStart int
	pageIds *[]int
	pageList *[]pgObj
	objList *[]pdfObj
	rdObjList *[]pdfObj
	fonts *[]objRef
	gStates *[]objRef
	xObjs *[]objRef
	fontAux *[]fontAuxObj
	mediabox *[4]float32
	test bool
	verb bool
}

type pdfObj struct {
	objId int
	typ int
	typstr string
	dict bool
	array bool
	simple bool
	names *[]string
	parent int
	start int
	end int
	contSt int
	contEnd int
	streamSt int
	streamEnd int
}

type pgObj struct {
	pageNum int
	mediabox *[4]float32
	contentId int
	parentId int
	fontId int
	gStateId int
	xObjId int
	fonts *[]objRef
	gStates *[]objRef
	xObjs *[]objRef
	fontAux *[]fontAuxObj
}

type objRef struct {
	Id int
	Nam string
}

type fontAuxObj struct {
	fontDesc int
	name string
	typ int
	glNum int
}

type gObj struct {
	gObjItem  objRef
}

type resourceList struct {
	fonts *[]objRef
	gStates *[]objRef
	xObjs *[]objRef
}

type pdfDoc struct {
	pageType int

}



func Init()(info *InfoPdf) {
	var pdf InfoPdf
	pdf.test = false
	pdf.verb = true
	return &pdf
}

func (pdf *InfoPdf) ReadPdf(parseFilnam string) (err error) {

   	parseFil, err := os.Open(parseFilnam)
    if err != nil {
        return fmt.Errorf("error opening file \"%s\"! %v\n", parseFilnam, err)
    }

    filinfo, err := parseFil.Stat()
    if err != nil {
	    return fmt.Errorf("error opening file \"%s\"! %v\n", parseFilnam, err)
    }


	pdf.fil = parseFil
	pdf.filSize = filinfo.Size()
	pdf.filNam = parseFilnam
	return nil
}


func (pdf *InfoPdf) parseTopTwoLines()(err error) {

	buf := *pdf.buf

	maxPos := 20
	if len(buf) < maxPos {maxPos = len(buf)}

	//read top line
	endFl := -1
	for i:=0; i<maxPos; i++ {
		if (buf[i] == '\n') {
			endFl = i
			break
		}
	}
	if endFl == -1 {return fmt.Errorf("no eol in first line!")}

	idx := bytes.Index(buf[:5],[]byte("%PDF-"))
	if idx == -1 {return fmt.Errorf("first line %s string is not \"%%PDF-\"!", string(buf[:5]))}

	verStr := string(buf[5:endFl])

	majver:= 99
	minver:= 99
	_, err = fmt.Sscanf(verStr, "%d.%d", &majver, &minver)
	if err != nil {return fmt.Errorf("cannot parse pdf version: %v!",err)}

	fmt.Printf("pdf version maj:min: %d:%d\n", majver, minver)

	if majver > 2 {return fmt.Errorf("invalid pdf version %d", majver)}

	pdf.majver = majver
	pdf.minver = minver

	// second line

	endSl := -1
	for i:=endFl+1; i<maxPos; i++ {
		if (buf[i] == '\n') {
			endSl = i
			break
		}
	}
	if endSl == -1 {return fmt.Errorf("no eol in second line!")}

	startSl := endFl +1
	dif := endSl - startSl
	if buf[endSl -1] == '\r' {dif--}
//fmt.Printf("2 line: %s\n", string(buf[endFl+1:endSl]))

	if dif != 5 {return fmt.Errorf(" no 4 chars after percent char %d:%d!", startSl, endSl)}

	for i:=startSl+1; i<startSl+5; i++ {
		if !(buf[i] > 120) {return fmt.Errorf("char %q not valid in second top line!", buf[i])}
	}

	return nil
}

func (pdf *InfoPdf) parseLast3Lines()(err error) {

	buf := *pdf.buf

	llEnd := len(buf) -1

	// now we have the real last line which should contain %%EOF
	slEnd := -1

	for i:=llEnd; i>llEnd -8; i-- {
		if buf[i] == '\n' {
			slEnd = i
			break
		}
	}

	if slEnd == -1 {return fmt.Errorf("cannot find eof for second top line")}

	idx := bytes.Index(buf[slEnd+1:], []byte("%%EOF"))
	if idx == -1 {return fmt.Errorf("last line: cannot find \"%%EOF\"!")}

	//let see wether there is a second line with eof

	sePos := slEnd - 500
	if sePos < 0 {sePos = 10}
	sidx := bytes.Index(buf[sePos:slEnd], []byte("%%EOF"))
	if sidx > 0 {
		fmt.Printf("found second \"%%EOF\"!\n")
		slEnd = sePos + sidx -1
	}

	//next we need to check second last line which should contain a int number

	tlEnd := -1
	for i:=slEnd-15; i<slEnd; i++ {
		if buf[i] == '\n' {
			tlEnd = i
			break
		}
	}
//fmt.Printf("tlEnd search: %s\n", string(buf[slEnd-15:slEnd]))
	if tlEnd == -1 {return fmt.Errorf("cannot find eof for third top line")}

	idx = bytes.Index(buf[tlEnd - 50:tlEnd], []byte("startxref"))
	if idx == -1 {return fmt.Errorf("last line: cannot find \"startxref\"!")}

	pdf.startxref = idx + tlEnd - 50
//fmt.Printf("startxref: %s\n", string(buf[pdf.startxref:tlEnd]))
	xref:=0
	_, err = fmt.Sscanf(string(buf[tlEnd+1:slEnd]),"%d", &xref)
	if err != nil {return fmt.Errorf("cannot parse xref: %v!", err)}
	pdf.xref = xref

	return nil
}

func (pdf *InfoPdf) parseTrailer()(err error) {

	if pdf.startxref < 1 {return fmt.Errorf("no valid startxref")}
	if pdf.trailer < 1 {return fmt.Errorf("no valid trailer")}

	buf := *pdf.buf
	trailEnd := 0
	for i:=pdf.startxref; i> pdf.startxref-10; i-- {
		if buf[i] == '>' {
			if buf[i-1] == '>' {
				trailEnd = i-1
				break
			}
		}
	}

	if trailEnd == 0 {
		return fmt.Errorf("cannot find closing angular bracket for trailer!")
	}

	trailStart := 0
	for i:=pdf.trailer + 7; i< trailEnd; i++ {
		if buf[i] == '<' {
			if buf[i+1] == '<' {
				trailStart = i+2
				break
			}
		}
	}

	if trailStart == 0 {
		outstr := "// cannot find opening angular brackets for trailer!"
		return fmt.Errorf(outstr)
	}

//	fmt.Printf("trailer: %s\n", string(buf[trailStart:trailEnd]))

	objId, err := pdf.parseObjRef("Root",trailStart, 1)
	if err != nil {return fmt.Errorf("parse Root Obj error: %v!", err)}
	pdf.rootId = objId
//fmt.Printf("Root: %d\n", objId)

	objId, err = pdf.parseObjRef("Info",trailStart, 1)
	if err != nil {return fmt.Errorf("parse Info Obj error: %v!", err)}
	pdf.infoId = objId

	objId, err = pdf.parseObjRef("Size",trailStart, 2)
	if err != nil {return fmt.Errorf("parse Size Obj error: %v!", err)}
	pdf.numObj = objId


	return nil
}



func (pdf *InfoPdf) parseObjRef(key string, Start int, Type int)(objId int, err error) {

	//find key
	buf := *pdf.buf
	keyByt := []byte("/" + key)
	rootSt:= -1

	linEnd := pdf.startxref
	if (Start + 400) < linEnd {linEnd = Start+400}


//fmt.Printf("trailer start: %d end: %d %s\n", Start, pdf.startxref, string(buf[Start:pdf.startxref]))

//fmt.Printf("key: %s\nstr [%d:%d]: %s\n", string(keyByt), Start, linEnd, string(buf[Start:(Start + 40)]))

	for i:=Start; i< linEnd; i++ {
		ires := bytes.Index(buf[Start: linEnd], keyByt)
		if ires > -1 {
			rootSt = Start + ires
			break
		}
	}
	if rootSt == -1 {return -1, fmt.Errorf("cannot find keyword %s", key)}

	valSt:= rootSt+len(keyByt)
	rootEnd := -1
	for i:=valSt; i< linEnd; i++ {
		switch buf[i] {
		case '/','\n','\r':
			rootEnd = i
			break
		default:
		}
	}

	if rootEnd == -1 {return -1, fmt.Errorf("cannot find end delimiter after key %s", key)}

	keyObjStr := string(buf[valSt:rootEnd])

//fmt.Printf("%s: %s\n", key, keyObjStr)

	switch Type {
	case 1:
		val :=0
		_, err = fmt.Sscanf(keyObjStr, " %d %d R", &objId, &val)
		if err != nil {return -1, fmt.Errorf("cannot parse obj ref after keyword %s! %v", key, err)}
	case 2:
		_, err = fmt.Sscanf(keyObjStr, " %d", &objId)
		if err != nil {return -1, fmt.Errorf("cannot parse obj val after keyword %s! %v", key, err)}
	default:
		return -1, fmt.Errorf("invalid obj value type!")
	}
	return objId, nil
}

func (pdf *InfoPdf) parseTrailCont(linStr string)(outstr string, err error) {

	keyStr := string(linStr[1:5])
	switch keyStr {
	case "Size":
		size:= 0
		_, err = fmt.Sscanf(string(linStr[5:]),"%d",&size)
		if err != nil {
			outstr += fmt.Sprintf("Size: could not parse Size value: %v", err)
			return outstr, fmt.Errorf("could not parse Size value: %v", err)
		}
		pdf.objSize = size
		outstr += fmt.Sprintf("  Size: %d parsed correctly\n", size)

	case "Info":
		objId := 0
		val := 0
		_, err = fmt.Sscanf(string(linStr[5:]),"%d %d R",&objId, &val)
		if err != nil {
			outstr += fmt.Sprintf("  Info: %s:: could not parse Info: %v", string(linStr[5:]), err)
			return outstr, fmt.Errorf("Info: could not parse value: %v", err)
		}
		pdf.infoId = objId
		outstr += fmt.Sprintf("  Info: objId: %d  ref: %d R parsed successfully\n", objId, val)
	case "Root":
		objId := 0
		val := 0
		_, err = fmt.Sscanf(string(linStr[5:]),"%d %d R",&objId, &val)
		if err != nil {
			outstr += fmt.Sprintf("  Root: %s:: could not parse Info: %v", string(linStr[5:]), err)
			return outstr, fmt.Errorf("Root: could not parse value: %v", err)
		}
		pdf.rootId = objId
		outstr += fmt.Sprintf("  Root: objId: %d  ref: %d R parsed successfully\n", objId, val)

	default:
		outstr = fmt.Sprintf("%s is not a recognized keyword in Trailer\n", keyStr)
		return outstr, fmt.Errorf("invalid key word: %s", keyStr)
	}
	return outstr, nil
}

/*
func (pdf *InfoPdf) getKVStr(instr string)(outstr string, err error) {

	stPos := -1
	endPos := 0

	for i:=0; i< len(instr); i++ {
		if instr[i] == '<' {
			if instr[i+1] == '<' {
				stPos = i+2
				break
			}
		}
	}

	if stPos == -1 {return "", fmt.Errorf("no open double bracket!")}

	for i:=len(instr)-1; i> stPos; i-- {
		if instr[i] == '>' {
			if instr[i-1] == '>' {
				endPos = i-1
				break
			}
		}
	}

	if endPos == 0 {return "", fmt.Errorf("no closing brackets!")}

	outstr = instr[stPos: endPos] + "\n"
	return outstr, nil
}

func (pdf *InfoPdf) getKvMap(instr string)(kvMap map[string]string , err error) {

//fmt.Printf("******* start getKvMap\n")
//fmt.Println(instr)
//fmt.Printf("******* end getKvMap\n")
	ist := 0
	icount := 0
	linStr := ""
	key := ""
	val := ""
	valStr := ""
	kvMap = make(map[string]string)

	for i:=0; i< len(instr); i++ {
		if instr[i] != '\n' { continue}

		linStr = instr[ist:i]
		icount++
//fmt.Printf("linStr %d: %s\n", icount, linStr)
			_, err = fmt.Sscanf(linStr, "/%s %s", &key, &val)
			if err != nil {return kvMap, fmt.Errorf("parse error in line %d %s %v", icount, linStr, err)}
			// 2 : first letter is / second is ws
//fmt.Printf("key: %s val: %s %q\n", key, val, val[0])

			switch val[0] {
			case '/':
				valStr = linStr[(len(key)+2):]
				ist = i+1

//fmt.Printf("    /valStr: %s\n", valStr)

			case '<':
				remSt := ist + len(key) +1
				remStr := string(instr[remSt:])
//fmt.Printf("remStr: %s %d\n", remStr, remSt)
				tvalStr, errStr := pdf.getKVStr(remStr)
				if errStr != nil {return kvMap, fmt.Errorf("parse error in line %s %v", remStr, errStr)}
//fmt.Printf("   <valStr: %s\n", valStr)
				ist = remSt + len(tvalStr) + 5
				i = ist + 6
				tvalByt := []byte(tvalStr)
				for j:=0; j< len(tvalByt); j++ {
					if tvalByt[j] == '\n' {tvalByt[j] = ' '}
				}
				valStr = string(tvalByt)
			default:
				valStr = linStr[(len(key)+2):]
				ist = i+1
//fmt.Printf("    def valStr: %q %s\n", val[0], valStr)

			}


			kvMap[key] = valStr

	} // i



	if ist == 0 {return kvMap, fmt.Errorf("no eol found!")}

	return kvMap, nil
}
*/

func (pdf *InfoPdf) decodeStream(objId int)(streamSl *[]byte, err error) {

//fmt.Printf("******* getstream instr\n")
//fmt.Println(instr)
//fmt.Printf("******* end ***\n")

	buf := *pdf.buf
	obj := (*pdf.objList)[objId]

	if obj.streamSt < 0 {return nil, fmt.Errorf("no stream start found!")}

	streamLen := obj.streamEnd -obj.streamSt
	if streamLen < 1 {return nil, fmt.Errorf("no stream found!")}

	if pdf.verb {fmt.Printf("**** stream [length: %d] ****\n", streamLen)}

	stbuf := buf[obj.streamSt:obj.streamEnd]

	bytStream := bytes.NewReader(stbuf)

	streamBuf := new(bytes.Buffer)

	bytR, err := zlib.NewReader(bytStream)
	if err != nil {return nil, fmt.Errorf("stream deflate error: %v", err)}

//	_ = copy(stream, bytR)
	io.Copy(streamBuf, bytR)

	bytR.Close()

	stream := streamBuf.Bytes()
	return &stream, nil
}

func (pdf *InfoPdf) parseXref()(err error) {

	var pdfobj pdfObj
	var pdfObjList []pdfObj

	buf := *pdf.buf

	pdf.objList = &pdfObjList


	endStr := ""
	linStr := ""
	ist := pdf.xref
	istate := 0
	objId := 0
	objNum := 0
	objCount := 0
	objSt := 0
	val2 := 0
	totObj :=0

	outstr :=""

	if pdf.xref <1 { return fmt.Errorf("not a valid xref supplied!")}
	if pdf.trailer <1 { return fmt.Errorf("not a valid trailer supplied!")}

	// check xref

	// replace with readLine ?
	linCount := 0
	for i:= ist; i < pdf.trailer; i++ {
		if buf[i] == '\n' {
			linCount++
			linEnd:= i
			if buf[i-1] == '\r' { linEnd--}
			if linEnd < ist {fmt.Printf("error linEnd\n")}
			linStr = string(buf[ist:linEnd])
			ist = i+1
		} else {
			continue
		}

//fmt.Printf("linCount: %d istate %d objCount: %d linStr: %s\n", linCount, istate, objCount, linStr)
		switch istate {
		case 0:
			if bytes.Index([]byte(linStr), []byte("xref")) > -1 {
				istate = 1
			} else {
				return fmt.Errorf("cannot find \"xref\"!")
			}
		case 1:
			_, err1 := fmt.Sscanf(linStr, "%d %d", &objId, &objNum)
			if err1 != nil {
				outstr = fmt.Sprintf(" error parsing expected object heading [objid num] %s: %v", linStr, err)
				return fmt.Errorf(outstr)
			}
			totObj += objNum
			istate = 2
		case 2:
			_, err = fmt.Sscanf(linStr, "%d %d %s", &objSt, &val2, &endStr)
			if err != nil {
				outstr = fmt.Sprintf("   //error parsing object %d: %v", objCount, err)
				return fmt.Errorf(outstr)
			}
			if objCount > objNum {
				outstr = fmt.Sprintf("   //error too many obj ref %d: %v", objCount, err)
				return fmt.Errorf(outstr)
			}

			pdfobj.objId = objId + objCount
			pdfobj.start = objSt
			if endStr == "f" {pdfobj.start = 0}
			pdfObjList = append(pdfObjList, pdfobj)
			if objCount == objNum {istate = 1}
			objCount++

		default:

		}

	} // i


	pdf.numObj = objCount

//fmt.Printf("num objs: %d\n", objCount)

	for i:=0; i< len(pdfObjList); i++ {
		pdfObjList[i].end = 0
		if pdfObjList[i].start == 0 {continue}
		objEnd := pdf.xref
		for j:= 0; j<len(pdfObjList); j++ {
			if (pdfObjList[j].start < objEnd) && (pdfObjList[j].start>pdfObjList[i].start) {
				objEnd = pdfObjList[j].start
			}
		}
		pdfObjList[i].end = objEnd
	}

	return nil
}

/*
func (pdf *InfoPdf) parseContent(instr string, pgNum int)(outstr string, err error) {

fmt.Println("**** Content ***")
	outstr = fmt.Sprintf("**** KV ****\n")

	objStr, err := pdf.getKVStr(instr)
	if err != nil {
		outstr += fmt.Sprintf("// getVkStr error: %v\n", err)
		return outstr, fmt.Errorf("getVkStr error: %v", err)
	}


//fmt.Println("***** objStr parsePage")
//fmt.Println(objStr)
//fmt.Println("***** end objstr")

	kvm, err := pdf.getKvMap(objStr)
	if err != nil {
		outstr += fmt.Sprintf("getVkMap error: %v", err)
		return outstr, fmt.Errorf("getVkMap error: %v", err)
	}


	for key, val := range kvm {
		outstr += fmt.Sprintf("key: %s value: %s\n", key, val)
		fmt.Printf("key: %s value: %s\n", key, val)
	}

fmt.Println("*** end Content kv ")

	streamStr, err := pdf.getStream(instr)
	if err != nil {
		outstr += streamStr + fmt.Sprintf("stream deflate error: %v\n", err)
		return outstr, fmt.Errorf("stream deflate error: %v", err)
	}
	if len(streamStr) == 0 {
		outstr += "no stream\n"
		return outstr, nil
	}

//	outstr += streamStr

fmt.Printf("stream length: %d\n", len(streamStr))
	outstr += fmt.Sprintf("**** stream [length: %d] ****\n", len(streamStr))

	stbuf := []byte(streamStr)

	bytStream := bytes.NewReader(stbuf)

	bytR, err := zlib.NewReader(bytStream)
	if err != nil {
		outstr += fmt.Sprintf("stream deflate error: %v\n", err)
		return outstr, fmt.Errorf("stream deflate error: %v", err)
	}
	nbuf := new(strings.Builder)
	_, err = io.Copy(nbuf, bytR)
	if err != nil {
		outstr += fmt.Sprintf("stream copy error: %v\n", err)
		return outstr, fmt.Errorf("stream copy error: %v", err)
	}

	bytR.Close()

	outstr += nbuf.String()

fmt.Printf("stream:\n%s\n****\n", nbuf.String())
fmt.Println("***** end streamstr")

	return outstr, nil
}


func (pdf *InfoPdf) CheckPdf(textFile string)(err error) {

	var outstr string

	txtFil, err := os.Create(textFile)
	if err != nil {return fmt.Errorf("error creating textFile %s: %v\n", textFile, err);}
	defer txtFil.Close()

	pdf.txtFil = txtFil
	buf := make([]byte,pdf.filSize)

	_, err = (pdf.fil).Read(buf)
	if err != nil {return fmt.Errorf("error Read: %v", err)}

//	buf := pdf.convertEOL(filbuf)

	// 40 character should be more than enough
	outstr , err = pdf.parseTopTwoLines(buf[:40])
	outstr = "**** top two lines ***\n" + outstr
	txtFil.WriteString(outstr)
	if err != nil {return fmt.Errorf("parseTopTwoLines: %v",err)}


	// last line
	// first get rid of empty lines at the end

	l3LinStr , err := pdf.parseLast3Lines(&buf)
	outstr = "**** last three lines ***\n"
	if err != nil {
		txtFil.WriteString(outstr + l3LinStr)
		return fmt.Errorf("parseLast3Lines: %v",err)
	}
	pEndStr := outstr + l3LinStr

	// find trailer

	err = pdf.parseTrailer()
	trailerStr := "**** trailer ****\n"
	if err != nil {
		txtFil.WriteString(trailerStr + pEndStr)
		return fmt.Errorf("parseLast3Lines: %v",err)
	}

	pEndStr = trailerStr + pEndStr

	// find beginning of xref section
	trailer_start := pdf.trailer

	xref :=0
	for i:=trailer_start -1 ; i>0; i-- {
		if (buf[i] == 'x') {
			if string(buf[i: i+4]) == "xref" {
				xref = i
				break
			}
		}
	}

	if xref ==0 {
		outstr = fmt.Sprintf("// cannot find xref beginning of line!\n") + outstr
		return fmt.Errorf("cannot find xref beginning of line!")
	}

	xrefStr := fmt.Sprintf("xref       // found keyword xref at %d\n", xref)

	pdf.xref = xref

//	pdfObjList, err := pdf.parseXref()

	outstr = xrefStr
//	outstr += objStr
	outstr += pEndStr
	txtFil.WriteString(outstr)

	// sort

	txtFil.WriteString("**************************\n")
	pdfObjList := *(pdf.objList)
	pdfObjStr := ""
	for j:=0; j<len(pdfObjList); j++ {
		pdfObjStr += fmt.Sprintf("Obj Id: %4d Start: %5d End: %5d\n", pdfObjList[j].objId, pdfObjList[j].start, pdfObjList[j].end)
	}
	txtFil.WriteString(pdfObjStr)


	// check for obj and endobj
	for i:=0; i< len(pdfObjList); i++ {
		stp := pdfObjList[i].start
		contSt :=0
		contEnd :=0
		linstr :=""
		for j:= stp; j< stp + 80; j++ {
			if buf[j] == '\n' {
				linstr = string(buf[stp: j])
				contSt = j+1
				break
			}
		}
		if len(linstr) == 0 {
			txtFil.WriteString(fmt.Sprintf("Obj %d has no lf\n",i))
			return fmt.Errorf("Obj %d has no eol",i)
		}

//fmt.Printf("obj %d start: %s\n", i, linstr)

		objId :=0
		val := 0
		_, err = fmt.Sscanf(linstr,"%d %d obj",&objId, &val)
		if err != nil {
			txtFil.WriteString(fmt.Sprintf("Obj %d cannot parse: %v\n", err))
			return fmt.Errorf("Obj %d cannot parse: %v", err)
		}
		if objId != pdfObjList[i].objId {
			txtFil.WriteString(fmt.Sprintf("no match of Obj %d and objId %d \n", objId, pdfObjList[i].objId))
			return fmt.Errorf("no match of Obj %d and objId %d \n", objId, pdfObjList[i].objId)
		}

		// endobj
		endp := pdfObjList[i].end
		linstr =""
		for j:= endp - 2 ; j> stp; j-- {
			if buf[j] == '\n' {
				linstr = string(buf[j+1: endp -1])
				contEnd = j
				break
			}
		}

//fmt.Printf("obj %d end: %d %d %s\n", i, contEnd, endp, linstr)

		if linstr != "endobj" {
			txtFil.WriteString(fmt.Sprintf("Obj %d has no \"endobj\" string: %s\n", i, linstr))
			return fmt.Errorf("Obj %d has no \"endobj\" string: %s", i, linstr)
		}
//		fmt.Printf("obj %d:\n%s\n", i, buf[contSt:contEnd])
		pdfObjList[i].contSt = contSt
		pdfObjList[i].contEnd = contEnd
		// check for angular brackes


	} //i obj

	pdfObjStr = "*******\n"
	for j:=0; j<len(pdfObjList); j++ {
		pdfObjStr += fmt.Sprintf("num: %3d Obj Id: %4d Start: %5d End: %5d Content St: %5d End: %5d\n", j, pdfObjList[j].objId, pdfObjList[j].start, pdfObjList[j].end, pdfObjList[j].contSt, pdfObjList[j].contEnd)
	}
	txtFil.WriteString(pdfObjStr)


	// info object
	id := pdf.infoId - 1
	hdStr := fmt.Sprintf("************ Info [Obj: %d] **************\n", id+1)
	txtFil.WriteString(hdStr)
	infoStr := string(buf[(pdfObjList[id].contSt+2):(pdfObjList[id].contEnd -2)]) + "\n"
	txtFil.WriteString(infoStr)
fmt.Printf("info:\n%s", infoStr)

	//ROOT
	id = pdf.rootId - 1
	hdStr = fmt.Sprintf("************ Root [Obj: %d] **************\n", id+1)
	txtFil.WriteString(hdStr)
	rootStr := string(buf[(pdfObjList[id].contSt+2):(pdfObjList[id].contEnd -2)]) + "\n"
	txtFil.WriteString(rootStr)
fmt.Printf("************ root **************\n%s", rootStr)

	kvmap, err := pdf.parseRootOld(rootStr)
	if err != nil {
		outstr = fmt.Sprintf("// error parsing Root: %v\n", err)
		txtFil.WriteString(outstr)
		return fmt.Errorf("error parsing Root: %v", err)
	}
	outstr = "  // ROOT properties\n"
	for key, val := range kvmap  {
		outstr += fmt.Sprintf("   %s: %s\n", key, val)
	}

	outstr += fmt.Sprintf("//Obj Root with id %d parsed successfully\n", pdf.rootId)
	txtFil.WriteString(outstr)

//fmt.Printf("Obj ROOT with Obj id %d parsed successfully\n", id)

	// Pages Obj

	id = pdf.pagesId -1
	hdStr = fmt.Sprintf("************ Pages [Obj: %d] **************\n", id+1)
	txtFil.WriteString(hdStr)

	pagesStr :=string(buf[(pdfObjList[id].contSt+2):(pdfObjList[id].contEnd -2)]) + "\n"
	txtFil.WriteString( "string: \n" + pagesStr + "***** end string\n")
fmt.Printf("************ pages **********\n")
fmt.Println(pagesStr)
	err = pdf.parsePages()
	if err != nil {
		outstr = fmt.Sprintf("// error parsing Pages: %v\n", err)
		txtFil.WriteString(outstr)
		return fmt.Errorf("error parsing Pages: %v", err)
	}

	outstr = fmt.Sprintf("Pages // Pages parsed successfully\n")
	outstr += fmt.Sprintf("page count: %d\n", pdf.pageCount)
	for i:=0; i< pdf.pageCount; i++ {
		outstr += fmt.Sprintf("page %d: id: %d\n",i+1 ,(*pdf.pageList)[i])
	}
	txtFil.WriteString(outstr)


	// Page
	for pg:=0; pg<(pdf.pageCount ); pg++ {

		id := (*pdf.pageIds)[pg]
		hdstr := fmt.Sprintf("************ Page %d [Obj: %d] **************\n", pg+1, id)
		txtFil.WriteString(hdstr)

		pageStr := string(buf[(pdfObjList[id].contSt):(pdfObjList[id].contEnd)]) + "\n"
		txtFil.WriteString(pageStr)

fmt.Println(pageStr)

fmt.Printf("******** page %d Obj %d *************\n%s\n**************end pageStr ********\n",pg +1, id +1, pageStr)

		err := pdf.parsePage(pg)
		if err != nil {
			outstr = fmt.Sprintf("// error parsing Page %d: %v\n",pg ,err)
			txtFil.WriteString(outstr)
			return fmt.Errorf("error parsing Page: %v", err)
		}
		pgObj := (*pdf.pageList)[pg]
		outstr = fmt.Sprintf("// Page %d parsed successfully\n",pg)
		txtFil.WriteString(outstr)


		// need to parse each Page
		id = pgObj.contentId
		hdstr = fmt.Sprintf("************ Content Page %d [Obj %d] **************\n", pg+1, id+1)
		txtFil.WriteString(hdstr)

//fmt.Printf("page %d: contentId: %d\n", pg, id)
		contentStr := string(buf[(pdfObjList[id].contSt):(pdfObjList[id].contEnd)]) + "\n"

		// seperate stream and kv pairs
		outstr, err = pdf.parseContent(contentStr, pg)
		if err != nil {
			outstr += fmt.Sprintf("// error parsing Content %d for Page %d: %v\n",id, pg ,err)
			txtFil.WriteString(outstr)
			return fmt.Errorf("error parsing Content %d for Page %d: %v", id, pg, err)
		}
		txtFil.WriteString(outstr)


	} // page

	return nil

}
*/


func (pdf *InfoPdf) DecodePdf()(err error) {

	var outstr string

	buf := make([]byte,pdf.filSize)

	_, err = (pdf.fil).Read(buf)
	if err != nil {return fmt.Errorf("error Read: %v", err)}

	pdf.buf = &buf

	//read top two lines
	txtstr, nextPos, err := pdf.readLine(0)
	if err != nil {
		txtstr = fmt.Sprintf("// read top line: %v", err)
		return fmt.Errorf(txtstr)
	}
	outstr = txtstr + "\n"

	txtstr, nextPos, err = pdf.readLine(nextPos)
	if err != nil {
		txtstr = fmt.Sprintf("// read second top line: %v", err)
		return fmt.Errorf(txtstr)
	}

	// read last three lines

	bufLen := len(buf)

	ltStart := bufLen - 30
	sByt := []byte("startxref")

	ires := bytes.Index(buf[ltStart:], sByt)
	if ires < 0 {
		txtstr = "cannot find \"startxref\"!"
		return fmt.Errorf(txtstr)
	}

	ltStart += ires
	pdf.startxref = ltStart
//	fmt.Printf("ires %d\n%s\n", ires, string(buf[ltStart:]))

	txtstr, nextPos, err = pdf.readLine(ltStart)
	if err != nil {
		txtstr = fmt.Sprintf("// read third last line: %v", err)
		return fmt.Errorf(txtstr)
	}
	outstr += txtstr + "\n"

//fmt.Printf("second last line!!\n%s\n", string(buf[nextPos:]))
	txtstr, nextPos, err = pdf.readLine(nextPos)
	if err != nil {
		txtstr = fmt.Sprintf("// read second last line: %v", err)
		return fmt.Errorf(txtstr)
	}

	xref := 0
	_, err = fmt.Sscanf(txtstr, "%d", &xref)
	if err != nil {
		errconvStr := fmt.Sprintf("could not convert %s into xref: %v", txtstr, err)
		return fmt.Errorf(errconvStr)
	}
	pdf.xref = xref

//fmt.Printf("xref: %d\n", xref)

	// last line
	txtstr = string(buf[nextPos:])


	// trailer
	tStart := ltStart - 200

	tres := bytes.Index(buf[tStart:ltStart], []byte("trailer"))
	if tres < 0 {
		txtstr = "cannot find \"trailer\"!"
		return fmt.Errorf(txtstr)
	}
	tStart += tres
	pdf.trailer = tStart

	txtstr, nextPos, err = pdf.readLine(tStart)
	if err != nil {
		txtstr = fmt.Sprintf("// read line with \"trailer\": %v", err)
		return fmt.Errorf(txtstr)
	}

	err = pdf.parseTrailer()
	if err != nil {
		txtstr = fmt.Sprintf("parseTrailor: %v", err)
		return fmt.Errorf(txtstr)
	}

	// trailer content

	xByt := []byte("xref")

	xres := bytes.Index(buf[pdf.xref:pdf.xref+7], xByt)
	if xres < 0 {
		txtstr = "cannot find \"xref\"!"
		return fmt.Errorf(txtstr)
	}

//fmt.Printf("xres: %d\n", xres)

	txtstr, nextPos, err = pdf.readLine(pdf.xref)
	if err != nil {
		txtstr = fmt.Sprintf("// read line with \"xref\": %v", err)
		return fmt.Errorf(txtstr)
	}

	// parse  Xref
	err = pdf.parseXref()
	if err != nil {return fmt.Errorf("parseXref: %v", err)}

	//list each object
	objList := *pdf.objList
	for i:=0; i< len(objList); i++ {
		_, err := pdf.decodeObjStr(i)
		if err != nil {
			txtstr = fmt.Sprintf("// getObjStr Objid %d: %v", i, err)
			return fmt.Errorf(txtstr)
		}
	}

fmt.Println("\n*** parse Pdf Tree ***")
	// create pdf dom tree
	err = pdf.parseRoot()
	if err != nil {return fmt.Errorf("parseRoot: %v", err)}
	fmt.Printf("** parsed Root successfully **\n")

	err = pdf.parsePages()
	if err != nil {return fmt.Errorf("parsePages: %v", err)}

	fmt.Printf("** parsed Pages successfully **\n")

	return nil
}

func (pdf *InfoPdf) DecodePdfToText(txtfil string)(err error) {
// method that decodes pdf file to text
// parsing sequence
// Step 1: top 2 lines
// Step 2: last 3 lines
// Step 3: trailer section
// Step 4: xref section
// Step 5: objs
// Step 6: parseInfo
// Step 6: parseRoot
// Step 7: parsePages
// Step 8: for each page: parsePage
// Step 9: for each page: parseContent

	var outstr string

	txtFil, err := os.Create(txtfil)
	if err != nil {return fmt.Errorf("error creating textFile %s: %v\n", txtfil, err);}
	defer txtFil.Close()

	pdf.txtFil = txtFil

	buf := make([]byte,pdf.filSize)
	pdf.buf = &buf

	_, err = (pdf.fil).Read(buf)
	if err != nil {return fmt.Errorf("error Read: %v", err)}



	//read first line
	txtstr, nextPos, err := pdf.readLine(0)
	if err != nil {
		txtstr = fmt.Sprintf("// read top line: %v", err)
		txtFil.WriteString(txtstr + "\n")
		return fmt.Errorf(txtstr)
	}
	outstr = txtstr + "\n"

	// second line
	txtstr, nextPos, err = pdf.readLine(nextPos)
	if err != nil {
		txtstr = fmt.Sprintf("// read second top line: %v", err)
		txtFil.WriteString(outstr + txtstr + "\n")
		return fmt.Errorf(txtstr)
	}

	pdf.objStart = nextPos

	outstr += txtstr + "\n"
	txtFil.WriteString("********* first two lines ***********\n")
	txtFil.WriteString(outstr)

	err = pdf.parseTopTwoLines()
	if err != nil {return fmt.Errorf("parseTopTwoLines: %v", err)}
	outstr = "parsed top two lines successfully\n"
	txtFil.WriteString(outstr)
	fmt.Printf("%s", outstr)
//fmt.Printf("**** first two lines ***\n%s\n",outstr)

	err = pdf.parseLast3Lines()
	if err != nil {return fmt.Errorf("parseLast3Lines: %v", err)}
	outstr = "parsed last three lines successfully\n"
	txtFil.WriteString(outstr)
	fmt.Printf("%s", outstr)

/*
	// read last three lines
	txtFil.WriteString("******** last three lines ***********\n")
	bufLen := len(buf)
	outstr = ""

	ltStart := bufLen - 30

	ires := bytes.Index(buf[ltStart:], []byte("startxref"))

	if ires < 0 {
		txtstr = "cannot find \"startxref\"!"
		txtFil.WriteString(txtstr + "\n")
		return fmt.Errorf(txtstr)
	}

	ltStart += ires
	pdf.startxref = ltStart
//fmt.Printf("ires %d\n%s\n", ires, string(buf[ltStart:]))

	txtstr, nextPos, err = pdf.readLine(ltStart)
	if err != nil {
		txtstr = fmt.Sprintf("// read third last line: %v", err)
		txtFil.WriteString(outstr + txtstr + "\n")
		return fmt.Errorf(txtstr)
	}
	outstr += txtstr + "\n"

//fmt.Printf("second last line [%d]:\n%s\n", nextPos, string(buf[nextPos:]))
	txtstr, nextPos, err = pdf.readLine(nextPos)
	if err != nil {
		txtstr = fmt.Sprintf("// read second last line: %v", err)
		txtFil.WriteString(outstr + txtstr + "\n")
		return fmt.Errorf(txtstr)
	}
	outstr += txtstr + "\n"
//fmt.Printf("end second line:\n%s\n", outstr)

//	pdf.test = false
	xref := 0
	_, err = fmt.Sscanf(txtstr, "%d", &xref)
	if err != nil {
		errconvStr := fmt.Sprintf("could not convert %s into xref: %v", txtstr, err)
		txtFil.WriteString(outstr + "error: " + errconvStr + "\n")
		return fmt.Errorf(errconvStr)
	}
	oldxref := xref
	pdf.xref = xref

	// last line
	outstr += string(buf[nextPos:])
	if buf[bufLen-1] != '\n' {outstr += "\n"}
	txtFil.WriteString(outstr)

	// id endobj above xref -> no other startxrefs
	ilast := bytes.Index(buf[xref - 10: xref], []byte("endobj"))
	// this is a hack but clearer than putting all the subsequent code into the if statement
	if ilast > 0 { goto parseTrailer}

	// let's check whether there are multiple startref

	txtFil.WriteString("found no \"endobj\" before \"xref\"\n")

	// check for a second key "startref

	ires = bytes.Index(buf[(xref - 30):xref], []byte("startxref"))
	if ires < 0 {
		txtstr = "cannot find second\"xref\": !"
		txtFil.WriteString(txtstr + "\n")
		return fmt.Errorf(txtstr)
	}


	pdf.startxref = xref -30 + ires

//fmt.Printf("xrefstart: %d\n%s\n", pdf.startxref, string(buf[pdf.startxref: ]))

	txtstr, nextPos, err = pdf.readLine(pdf.startxref)
	if err != nil {
		txtstr = fmt.Sprintf("// read second: third last line from EOF: %v", err)
		txtFil.WriteString(outstr + txtstr + "\n")
		return fmt.Errorf(txtstr)
	}
	outstr = txtstr + "\n"

//fmt.Printf("second: top last line:\n%s\n",outstr)

	txtstr, nextPos, err = pdf.readLine(nextPos)
	if err != nil {
		txtstr = fmt.Sprintf("// read second: second last line: %v", err)
		txtFil.WriteString(outstr + txtstr + "\n")
		return fmt.Errorf(txtstr)
	}
	outstr += txtstr + "\n"

//fmt.Printf("second: top 2 last lines:\n%s\n",outstr)

	_, err = fmt.Sscanf(txtstr, "%d", &xref)
	if err != nil {
		errconvStr := fmt.Sprintf("second xref: could not convert %s into xref: %v", txtstr, err)
		txtFil.WriteString(outstr + "error: " + errconvStr + "\n")
		return fmt.Errorf(errconvStr)
	}

    ilast = bytes.Index(buf[(xref-30):xref], []byte("endobj"))

    if ilast < 0 {
        txtstr = "cannot find any \"endobj\": invalid pdf!"
        txtFil.WriteString(txtstr + "\n")
        return fmt.Errorf(txtstr)
    }

	pdf.xref = xref

	outstr += string(buf[nextPos:oldxref])
//fmt.Printf("second: top last 3 lines:\n%s\n",outstr)

	txtFil.WriteString("********** second Last 3 Lines **********\n")
	txtFil.WriteString(outstr)
//fmt.Printf("**** last three lines ***\n%s\n",outstr)

// after checking for multiple endings we can parse the correct trailer section
parseTrailer:
	//trailer
*/
	txtFil.WriteString("******** trailer ***********\n")

	outstr = ""
	tStart := pdf.startxref - 200
	tres := bytes.Index(buf[tStart:pdf.startxref], []byte("trailer"))
	if tres < 0 {
		txtstr = "cannot find \"trailer\"!"
		txtFil.WriteString(txtstr + "\n")
		return fmt.Errorf(txtstr)
	}

	tStart += tres
	pdf.trailer = tStart

	txtstr, nextPos, err = pdf.readLine(tStart)
	if err != nil {
		txtstr = fmt.Sprintf("// read line with \"trailer\": %v", err)
		txtFil.WriteString(outstr + txtstr + "\n")
		return fmt.Errorf(txtstr)
	}

	// txtstr should be "trailer"
	txtFil.WriteString(txtstr + "\n")

	// parses for root, info and size
	err = pdf.parseTrailer()
	if err != nil {
		txtstr = fmt.Sprintf("parse error \"trailer\": %v", err)
		txtFil.WriteString(txtstr + "\n")
		return fmt.Errorf(txtstr)
	}

	// trailer content
	txtstr = string(buf[nextPos:pdf.startxref])
	txtFil.WriteString(txtstr)

	// xref section
	txtFil.WriteString("******** xref ***********\n")
	outstr = ""
	xByt := []byte("xref")

	xres := bytes.Index(buf[pdf.xref:pdf.xref+7], xByt)
	if xres < 0 {
		txtstr = "cannot find \"xref\"!"
		txtFil.WriteString(txtstr + "\n")
		return fmt.Errorf(txtstr)
	}

//fmt.Printf("xres: %d\n", xres)

	txtstr, nextPos, err = pdf.readLine(pdf.xref)
	if err != nil {
		txtstr = fmt.Sprintf("// read line with \"xref\": %v", err)
		txtFil.WriteString(outstr + txtstr + "\n")
		return fmt.Errorf(txtstr)
	}

	txtFil.WriteString(txtstr + "\n")

	// xref content
	txtstr = string(buf[nextPos:tStart])

	txtFil.WriteString(txtstr)

	// parse  Xref
	err = pdf.parseXref()
	if err != nil {return fmt.Errorf("parseXref: %v", err)}

	// pdf body consisting of a list of "obj"
	txtFil.WriteString("******** sequential obj list ***********\n")

	// get Objects directly
	rdObjList, err := pdf.readObjList()
	if err != nil {return fmt.Errorf("readObjList: %v", err)}
	pdf.rdObjList  = rdObjList
	txtFil.WriteString("Obj ObjId  start  end\n")
	for i:=0; i< len(*rdObjList); i++ {
		obj := (*rdObjList)[i]
		outstr = fmt.Sprintf("%3d [%3d]: %5d %5d\n", i, obj.objId, obj.start, obj.end)
		txtFil.WriteString(outstr)
	}

	objList := *pdf.objList

	//list each object
	for i:=0; i< len(objList); i++ {
		outstr = fmt.Sprintf("\n******** Obj %d ***********\n", i)
		txtFil.WriteString(outstr)

		objstr, err := pdf.decodeObjStr(i)
		if err != nil {
			txtstr = fmt.Sprintf("// getObjStr %d: %v", i, err)
			txtFil.WriteString(txtstr + "\n")
			return fmt.Errorf(txtstr)
		}
		txtFil.WriteString(objstr)
	}

	// collect objecs from xref
	txtFil.WriteString("******** xref obj list ***********\n")
	txtFil.WriteString("                             Content      Stream\n")
	txtFil.WriteString("Obj   Id type start  end   Start  End  Start  End  Length\n")
	for i:= 0; i< len(*pdf.objList); i++ {
		obj := (*pdf.objList)[i]
		outstr = fmt.Sprintf("%3d: %3d  %2d  %5d %5d %5d %5d %5d %5d %5d   %-15s\n",
		i, obj.objId, obj.typ, obj.start, obj.end, obj.contSt, obj.contEnd, obj.streamSt, obj.streamEnd, obj.streamEnd - obj.streamSt, obj.typstr)
		txtFil.WriteString(outstr)
	}
	txtFil.WriteString("\n")

fmt.Println("\n*** parsing Pdf Tree ***\n")
	// create pdf dom tree


fmt.Println("\n****** parsing Obj \"Catalog\" ******\n")
	err = pdf.parseRoot()
	if err != nil {return fmt.Errorf("parseRoot: %v", err)}
	txtFil.WriteString("parsed \"Root\" successfully!\n")
fmt.Printf("*** parsed Obj \"Catalog\" successfully ***\n")

fmt.Printf("\n******* parsing Obj \"Pages\" *******\n")
	err = pdf.parsePages()
	if err != nil {return fmt.Errorf("parsePages: %v", err)}
	txtFil.WriteString("parsed \"Pages\" successfully!\n")
fmt.Printf("***** parsed Pages successfully ******\n")
fmt.Println()

	// parsing each Page Obj
	for ipg:=0; ipg< pdf.pageCount; ipg++ {
		fmt.Printf("******* parsing Obj \"Page %d\" *******\n", ipg + 1)
		err = pdf.parsePage(ipg)
		if err != nil {return fmt.Errorf("parsePage %d: %v",ipg, err)}
		outstr := fmt.Sprintf("parsed \"Page %d\" successfully!", ipg)
		txtFil.WriteString(outstr +"\n")
		fmt.Printf("***** parsed %s ******\n", outstr)

		fmt.Printf("\n***** parsing Content of Page %d *******\n", ipg+1)
		err = pdf.parsePageContent(ipg)
		if err != nil {return fmt.Errorf("parsePageContent %d: %v",ipg, err)}
		outstr = fmt.Sprintf("parsed Content successfully!")
		txtFil.WriteString(outstr +"\n")
		fmt.Printf("****** %s ********\n", outstr)

		fmt.Println()
	}

	return nil
}

func (pdf *InfoPdf) readObjList()(rdobjList *[]pdfObj, err error) {

	var obj pdfObj
	var objList []pdfObj

	buf := *pdf.buf

	objSt:= pdf.objStart

	for i:=0;i<pdf.numObj; i++ {
		obj.start = objSt

//		if objIdx == -1 {return nil, fmt.Errorf("cannot find \"obj\" for obj %d of %d!", i, pdf.numObj)}

		txtstr, nextPos, err := pdf.readLine(objSt)
		if err != nil {return nil, fmt.Errorf("readLine error for obj %d: %v", i, err)}
		if len(txtstr) < 2 {
			objSt = nextPos
			txtstr, nextPos, err = pdf.readLine(objSt)
		}

		objIdx := bytes.Index(buf[objSt: objSt+30], []byte("xref"))
		if objIdx > -1 {break}

//		objIdx2 := bytes.Index(buf[objSt: objSt+30], []byte("obj")
//		if objIdx == -1 {continue}

		objId:= -1
		fmt.Sscanf(txtstr,"%d 0 obj", &objId)
		if objId == -1 {return nil, fmt.Errorf("obj %d of %d no obj ref in \"%s\" found!", i, pdf.numObj, txtstr)}

		idx := bytes.Index(buf[objSt:pdf.xref],[]byte("endobj"))
		if idx == -1 {return nil, fmt.Errorf("no endobj in obj %d!",i+1)}

		txtstr, nextPos, err = pdf.readLine(objSt +idx + 6)
		if err != nil {return nil, fmt.Errorf("readLine \"endobj\" for obj %d: %v", i, err)}

		objSt = nextPos

		obj.objId = objId
		obj.end = objSt + idx + 6
		objList = append(objList, obj)
	}

//fmt.Printf("objs: %d\n",len(objList) )

	return &objList, nil
}


func (pdf *InfoPdf) parseObjLin(linSl []byte, istate int )(newstate int, err error) {

	var pdfobj pdfObj
	var objList []pdfObj

//fmt.Println("line: ",string(linSl))
	if (pdf.rdObjList) != nil {
		objList = *pdf.rdObjList
	}

	cObjIdx := len(objList) -1

	nxtLin := false
	for is:= 0; is<5; is++ {

		switch istate {
		case 0:
		// obj begin
			id:=0
			_, scerr := fmt.Sscanf(string(linSl), "%d 0 obj", &id)
			if scerr != nil {return istate, fmt.Errorf("obj start scan %s error: %v", string(linSl), scerr)}
			pdfobj.objId = id
			objList = append(objList, pdfobj)
			istate++
			nxtLin = true
		case 1:
		// obj type
			idx := bytes.Index(linSl, []byte("/Type"))
			if idx == -1 {
				objList[cObjIdx].typ = Data
			} else {
				tpos :=0
				for i:= 5; i< len(linSl);i++ {
					if linSl[i] == '/' {
						tpos = i
						break
					}
				}
				if tpos ==0 {return 0, fmt.Errorf("no type property found in %s", string(linSl))}
				tepos := len(linSl)-1
				for i:= tpos + 1; i< len(linSl);i++ {
					if linSl[i] == '/' || linSl[i] == ' ' {
						tepos = i
						break
					}
				}
				objTyp := string(linSl[tpos+1: tepos])
fmt.Printf("obj %d id %d type %s\n", cObjIdx, objList[cObjIdx].objId, objTyp)
			}
			istate++
			nxtLin = true
		case 2:
		// stream
			istate = 4
			nxtLin = false
		case 3:
		// endstream

		case 4:
		// end obj
			idx := bytes.Index(linSl, []byte("endobj"))
//return 0, fmt.Errorf("no endobj found in %s", string(linSl))}

			nxtLin = true
			if idx> -1 {istate = 0}
		default:
			return istate, fmt.Errorf("invalid istate: %d", istate)
		}
		if nxtLin {break}
	} // is
	pdf.rdObjList = &objList
	newstate = istate
	return newstate, nil
}

func (pdf *InfoPdf) parseRoot()(err error) {

	if pdf.rootId > pdf.numObj {return fmt.Errorf("invalid rootId!")}
	if pdf.rootId ==0 {return fmt.Errorf("rootId is 0!")}

	obj := (*pdf.objList)[pdf.rootId]

	objId, err := pdf.parseObjRef("Pages",obj.contSt, 1)
	if err != nil {return fmt.Errorf("Root obj: parsing name \"/Pages\" error: %v!", err)}

	pdf.pagesId = objId

	outstr, err := pdf.parseKeyText("Title", obj)
	if err != nil {}
//return fmt.Errorf("Root obj: parsing name \"/Title\" error: %v!", err)}

fmt.Printf("Title: %s\n", outstr)
	return nil
}


func (pdf *InfoPdf) parseKeyText(key string, obj pdfObj)(outstr string, err error) {

	buf:= *pdf.buf
	objByt := buf[obj.contSt:obj.contEnd]
//fmt.Printf("found key: %s in %s\n", key, string(objByt))

	keyByt := []byte("/" + key)
	ipos := bytes.Index(objByt, keyByt)
	if ipos == -1 {return fmt.Sprintf("no /%s",key), fmt.Errorf("could not find keyword \"%s\"!", key)}

	valSt:= obj.contSt + ipos + len(keyByt) + 1
	rootEnd := -1
	for i:=valSt; i< obj.contEnd; i++ {
		switch buf[i] {
		case '/','\n','\r':
			rootEnd = i
			break
		default:
		}
	}

	if rootEnd == -1 {return fmt.Sprintf("/%s no value", key), fmt.Errorf("cannot find end delimiter after key %s", key)}

	outstr = string(buf[valSt:rootEnd])

	return outstr, nil
}

//page
func (pdf *InfoPdf) parsePage(iPage int)(err error) {

	var pgobj pgObj
//	buf := *pdf.buf

	pgobj.pageNum = iPage

	//determine the object id for page iPage
	pgobjId := (*pdf.pageIds)[iPage]
	txtFil := pdf.txtFil

	outstr := fmt.Sprintf("***** Page %d: id %d *******\n", iPage+1, pgobjId)
	fmt.Printf(outstr)
	txtFil.WriteString(outstr)

	// get obj for page[iPage]
	obj := (*pdf.objList)[pgobjId]

//fmt.Printf("testing page %d string:\n%s\n", iPage+1, string(buf[obj.start: obj.end]))

	objId, err := pdf.parseObjRef("Contents",obj.contSt, 1)
	if err != nil {return fmt.Errorf("parse \"Contents\" error: %v!", err)}

	outstr = fmt.Sprintf("Contents Obj: %d\n", objId)
	fmt.Printf(outstr)
	txtFil.WriteString(outstr)

	pgobj.contentId = objId

	mbox, err := pdf.parseMbox(obj)
	if err!= nil {
		pdf.txtFil.WriteString("no Name \"/MediaBox\" found!\n")
		fmt.Println("no Name \"/MediaBox\" found!")
	} else {
		pgobj.mediabox = mbox
		outstr = fmt.Sprintf("MediaBox: %.1f %.1f %.1f %.1f\n", mbox[0], mbox[1], mbox[2], mbox[3])
		txtFil.WriteString(outstr)
		fmt.Println(outstr)
	}

	reslist, err := pdf.parseResources(obj)
//fmt.Printf("page %d reslist: %v\n", iPage, reslist)
	if err != nil {
		outstr := fmt.Sprintf("parsing error \"/Resources\": %v!\n", err)
		pdf.txtFil.WriteString(outstr)
		fmt.Println(outstr)
	}

	if reslist != nil {
		if reslist.fonts != nil {pgobj.fonts = reslist.fonts}
		if reslist.gStates != nil {pgobj.gStates = reslist.gStates}
		if reslist.xObjs != nil {pgobj.xObjs = reslist.xObjs}
	}

//fmt.Printf("page %d: %v\n", iPage, pgobj)
	(*pdf.pageList)[iPage] = pgobj

	return nil
}


func (pdf *InfoPdf) parsePageContent(iPage int)(err error) {

	pgobj := (*pdf.pageList)[iPage]

	contId := pgobj.contentId

fmt.Printf("content obj: %d\n", contId)

	obj := (*pdf.objList)[contId]

	buf := *pdf.buf
fmt.Printf("cont obj [%d:%d]:\n%s\n", obj.contSt, obj.contEnd, string(buf[obj.contSt:obj.contEnd]))

	txtFil := pdf.txtFil

	outstr := fmt.Sprintf("***** Content Page %d: id %d *******\n", iPage+1, contId)

	fmt.Printf(outstr)
	txtFil.WriteString(outstr)

	//Filter
	dictByt := buf[obj.contSt+2: obj.contEnd-2]
	key:="/Filter"
	idx := bytes.Index(dictByt, []byte(key))
	if idx == -1 {return fmt.Errorf("no keyword \"%s\" found!", key)}

	istate :=0
	posSt := -1
	posEnd := -1
fmt.Printf("dicByt val: %s\n", string(dictByt[idx+len(key):]))
	for i:=idx + len(key); i< len(dictByt); i++ {
		switch istate {
		case 0:
			if dictByt[i] == '/' {istate++; posSt=i}
		case 1:
			if dictByt[i] =='/' || dictByt[i] == ' ' {posEnd = i}
			if dictByt[i] =='\r' || dictByt[i] == '\n' {posEnd = i}
		default:
		}
		if posEnd > -1 {break}
	}
	if posSt == -1 {return fmt.Errorf("keyword %s has no value!", key)}
	if posEnd == -1 {posEnd = len(dictByt)}

	valstr := dictByt[posSt:posEnd]

	fmt.Printf("%s: %s\n", key, valstr)

	key = "/Length"
	idx = bytes.Index(dictByt, []byte(key))
	if idx == -1 {return fmt.Errorf("no keyword \"%s\" found!", key)}

	istate =0
	posSt = -1
	posEnd = -1
fmt.Printf("dictByt val: %s\n", string(dictByt[idx+ len(key):]))
	for i:=idx + len(key); i< len(dictByt); i++ {
		switch istate {
		case 0:
			if dictByt[i] == ' ' {istate++; posSt=i}
		case 1:
			if dictByt[i] =='/' || dictByt[i] == ' ' {posEnd = i}
			if dictByt[i] =='\r' || dictByt[i] == '\n' {posEnd = i}
		default:
		}
		if posEnd > -1 {break}
	}
	if posSt == -1 {return fmt.Errorf("keyword %s has no value!", key)}
	if posEnd == -1 {posEnd = len(dictByt)}

	valstr = dictByt[posSt:posEnd]

	fmt.Printf("%s: %s\n", key, valstr)


	return nil
}

func (pdf *InfoPdf) parsePages()(err error) {

	if pdf.pagesId > pdf.numObj {return fmt.Errorf("invalid pagesId!")}
	if pdf.pagesId ==0 {return fmt.Errorf("pagesId is 0!")}

	obj := (*pdf.objList)[pdf.pagesId]

//fmt.Printf("pages:\n%s\n", string(buf[obj.start: obj.end]))

	err = pdf.parseKids(obj)
	if err!= nil {return fmt.Errorf("parseKids: %v", err)}

//fmt.Printf("pages: pageCount: %d\n", pdf.pageCount)
	pageList := make([]pgObj, pdf.pageCount)

	mbox, err := pdf.parseMbox(obj)
	if err!= nil {
		pdf.txtFil.WriteString("no Name \"/MediaBox\" found!\n")
		fmt.Println("no Name \"/MediaBox\" found!")
	}
	pdf.mediabox = mbox

	reslist, err := pdf.parseResources(obj)
	if err!= nil {
		pdf.txtFil.WriteString("no Name \"/Resources\" found!\n")
		fmt.Println("no Name \"/Resources\" found!")
	}

//fmt.Printf("resList: %v\n", reslist)

	if reslist != nil {
		if reslist.fonts != nil {pdf.fonts = reslist.fonts}
		if reslist.gStates != nil {pdf.gStates = reslist.gStates}
		if reslist.xObjs != nil {pdf.gStates = reslist.xObjs}
	}
	pdf.pageList = &pageList
	return nil
}

//rrr
func (pdf *InfoPdf) parseResources(obj pdfObj)(resList *resourceList, err error) {

	var reslist resourceList

	buf := *pdf.buf
	objByt := buf[obj.contSt:obj.contEnd]
//fmt.Printf("Resources: %s\n",string(objByt))

	idx := bytes.Index(objByt, []byte("/Resources"))
	if idx == -1 {
		if pdf.verb {fmt.Printf("parseResource: cannot find keyword \"/Resources\"!\n")}
		return nil, nil
	}

	// either indirect or a dictionary
	valst := obj.contSt + idx + len("/Resources")
	objByt = buf[valst: obj.contEnd]
//fmt.Printf("Resources valstr [%d:%d]: %s\n",valst, obj.contEnd, string(objByt))

	dictSt := bytes.Index(objByt, []byte("<<"))
//fmt.Printf("dictSt: %d\n", dictSt)

	if dictSt == -1 {
//fmt.Println("Resources: indirect obj")
		valend := -1
		for i:= valst; i< obj.contEnd; i++ {
			if buf[i] == 'R' {
				valend = i+1
				break
			}
		}
		if valend == -1 {return nil, fmt.Errorf("cannot find R for indirect obj of \"/Resources\"")}
		inObjStr := string(buf[valst:valend])

//fmt.Printf("ind obj: %s\n", inObjStr)

		objId :=0
		rev := 0
		_, err = fmt.Sscanf(inObjStr,"%d %d R", &objId, &rev)
		if err != nil{return nil, fmt.Errorf("cannot parse %s as indirect obj of \"/Resources\": %v", inObjStr, err)}

		fmt.Printf("Resource Id: %d\n", objId)
//todo find resource string
		return nil, nil
	}

	if pdf.verb {fmt.Println("**** Resources: dictionary *****")}

	resByt := buf[valst: obj.contEnd -2]
//fmt.Printf("Resources valstr [%d:%d]: %s\n",valst, obj.contEnd-2, string(resByt))

	// short cut to be fixed by parsing nesting levels
	dictEnd := bytes.LastIndex(resByt, []byte(">>"))
	if dictEnd == -1 {return nil, fmt.Errorf("no end brackets for dict!")}

	tByt := buf[valst: valst+dictEnd-2]
	tEnd := bytes.LastIndex(tByt, []byte(">>"))
	tSt := bytes.LastIndex(tByt, []byte("<<"))
	if tEnd < tSt {dictEnd = tEnd}

	if dictEnd == -1 {return nil, fmt.Errorf("no end brackets for dict!")}


	dictSt += valst +2
	dictEnd += valst
	dictByt := buf[dictSt:dictEnd]
//fmt.Printf("Resource Dict [%d: %d]:\n%s\n", dictSt, dictEnd, string(dictByt))
//fmt.Println()

	// find Font
	if pdf.verb {fmt.Println("**** Font: dictionary *****")}
	fidx := bytes.Index(dictByt, []byte("/Font"))
	if fidx == -1 {
		if pdf.verb {fmt.Println("parseResources: no keyword \"/Font\"!")}
		reslist.fonts = nil
	} else {
		objrefList, objId, err := pdf.parseIObjRefList("Font", &dictByt)
		if err != nil {return nil, fmt.Errorf("cannot get objList for \"/Font\": %v!", err)}
		reslist.fonts = objrefList
		if pdf.verb {fmt.Printf("fonts [%d]: %v\n",objId ,objrefList)}
	}
fmt.Printf("reslist: %v\n", reslist)

	// ExtGState
	if pdf.verb {fmt.Println("**** ExtGstate: dictionary *****")}
	gidx := bytes.Index(dictByt, []byte("/ExtGState"))
	if gidx == -1 {
		if pdf.verb {fmt.Println("parseResources: no keyword \"/ExGState\"!")}
		reslist.gStates = nil
	} else {
		objrefList, objId, err := pdf.parseIObjRefList("ExtGState", &dictByt)
		if err != nil {return nil, fmt.Errorf("cannot get objList for \"/ExtGState\": %v!", err)}
		reslist.gStates = objrefList
		if pdf.verb {fmt.Printf("ExtGState [%d]: %v\n",objId ,objrefList)}
	}

fmt.Printf("reslist: %v\n", reslist)


	// find XObject
	if pdf.verb {fmt.Println("**** XObject: dictionary *****")}
	xidx := bytes.Index(dictByt, []byte("/XObject"))
	if xidx == -1 {
		if pdf.verb {fmt.Println("parseResources: no keyword \"/XObject\"!")}
		reslist.xObjs = nil
	} else {
		objrefList, objId, err := pdf.parseIObjRefList("XObject", &dictByt)
		if err != nil {return nil, fmt.Errorf("cannot get objList for \"/XObject\": %v!", err)}
		reslist.xObjs = objrefList
		if pdf.verb {fmt.Printf("XObject [%d]: %v\n",objId ,objrefList)}
	}

fmt.Printf("reslist: %v\n", reslist)

	// ProcSet
fmt.Println("\n**** ProcSet: array *****")

	pidx := bytes.Index(dictByt, []byte("/ProcSet"))
	if pidx == -1 {return nil, fmt.Errorf("cannot find keyword \"/ProcSet\"")}


	pvalst := dictSt + pidx + len("/ProcSet")
	pByt := buf[pvalst: dictEnd]
//fmt.Printf("ProcSet valstr [%d:%d]: %s\n",pvalst, dictEnd, string(pByt))

	parrSt := bytes.Index(pByt, []byte("["))
//fmt.Printf("font dictSt: %d\n", fdictSt)

	parrEnd := bytes.Index(pByt, []byte("]"))
	if parrEnd == -1 {return nil, fmt.Errorf("no end bracket for ProcSet array!")}

	parrSt += pvalst +1
	parrEnd += pvalst
	parrByt := buf[parrSt:parrEnd]
fmt.Printf("ProcSet Array [%d: %d]: %s\n", parrSt, parrEnd, string(parrByt))

	return &reslist, nil
}


func (pdf *InfoPdf) parseIObjRefList(keyname string, dictbyt *[]byte)(objlist *[]objRef, objId int, err error) {

	var keyDictByt []byte
	dictByt := *dictbyt
	objId = -1

//fmt.Printf("****  parsing dictionary for %s *****\n", keyname)
//fmt.Printf("dict:\n%s\n", string(dictByt))

	keyByt := []byte("/" + keyname)
	fidx := bytes.Index(dictByt, []byte(keyByt))
	if fidx == -1 {return nil, -1, fmt.Errorf("cannot find keyword \"/%s\"",keyname)}

	dictEnd := len(dictByt)
	fvalst := fidx + len(keyByt)

	valByt := dictByt[fvalst: dictEnd]
//fmt.Printf("font valstr [%d:%d]: %s\n",fvalst, dictEnd, string(valByt))

	fdictSt := bytes.Index(valByt, []byte("<<"))
//fmt.Printf("font dictSt: %d\n", fdictSt)

	if fdictSt == -1 {
		if pdf.verb {fmt.Println("Resources: indirect obj")}
		fvalend := -1
		for i:= fvalst; i< dictEnd; i++ {
			if valByt[i] == 'R' {
				fvalend = i+1
				break
			}
		}
		if fvalend == -1 {return nil, -1, fmt.Errorf("cannot find R for indirect obj of \"/%s\"", keyname)}
		inObjStr := string(dictByt[fvalst:fvalend])

//fmt.Printf("ind obj: %s\n", inObjStr)

		rev := 0
		_, err = fmt.Sscanf(inObjStr,"%d %d R", &objId, &rev)
		if err != nil{return nil, -1, fmt.Errorf("cannot parse %s as indirect obj of \"/%s\": %v", inObjStr, keyname, err)}

		if pdf.verb {fmt.Printf("%s indirect Obj Id: %d\n", keyname, objId)}

		objSl, err := pdf.getObjCont(objId)
		if err != nil{return nil, -1, fmt.Errorf("cannot get content of obj %d: %v", objId, err)}

		keyDictByt = *objSl

	} else {

		fdictEnd := bytes.Index(valByt, []byte(">>"))
		if fdictEnd == -1 {return nil, -1, fmt.Errorf("no end brackets for dict of %s!", keyname)}

		keyDictByt = valByt[fdictSt +2 :fdictEnd]
	}

//fmt.Printf("%s key Dict: %s\n", keyname, string(keyDictByt))

	objList, err := pdf.parseIrefCol(&keyDictByt)
	if err != nil {return nil, objId, fmt.Errorf("%s parsing ref objs error: %v", keyname, err)}

	return objList, objId, nil
}

func (pdf *InfoPdf) parseIrefCol(inbuf *[]byte)(refList *[]objRef, err error) {

	var objref objRef
	var reflist []objRef

	buf := *inbuf

	val := 0
	objId := -1

	refCount := 0
	istate := 0

	objEnd := -1
	namSt:= -1
	namEnd := -1

	for i:= 0; i< len(buf); i++ {

		switch istate {
		case 0:
		// look for start of obj name
			if buf[i] == '/' {
				namSt = i+1
				istate = 1
			}
		case 1:
		// look for end of obj name
			if buf[i] == ' ' {
				namEnd = i
				istate = 2
			}

		case 2:
		// look for end of obj reference
			if buf[i] == 'R' {
				objEnd = i+1
				refCount++
//fmt.Printf(" inobjref: \"%s\"\n", string(buf[namEnd:objEnd]))
				_, errsc := fmt.Sscanf(string(buf[namEnd:objEnd])," %d %d R", &objId, &val)
				if errsc != nil {return nil, fmt.Errorf("parse obj ref error of obj %d: %v", refCount, errsc)}

				if namEnd< namSt {return nil, fmt.Errorf("parse obj name error of obj %d!", refCount)}
				objref.Id = objId
				objref.Nam = string(buf[namSt:namEnd])
				reflist = append(reflist, objref)
				istate = 0
			}
		default:
		}
	}


	return &reflist, nil
}

//obj
func (pdf *InfoPdf) getObjCont(objId int)(objSlice *[]byte, err error) {

	if objId > pdf.numObj {return nil, fmt.Errorf("objIs %d is not valid!", objId)}

	buf := *pdf.buf
	obj := (*pdf.objList)[objId]

	objByt := buf[obj.contSt:obj.contEnd]

	return &objByt, nil
}

func (pdf *InfoPdf) parseKids(obj pdfObj)(err error) {

	buf := *pdf.buf
	objByt := buf[obj.contSt:obj.contEnd]
//fmt.Printf("Kids: %s\n",string(objByt))

	idx := bytes.Index(objByt, []byte("/Kids"))
	if idx == -1 {return fmt.Errorf("cannot find keyword \"/Kids\"")}

	opPar := -1
	opEnd := -1

//fmt.Printf("brack start: %s\n", string(buf[(obj.contSt +5 + idx):obj.contEnd]))

	fini := false
	for i:= obj.contSt +idx + 5; i< obj.contEnd; i++ {
		switch buf[i] {
		case '[':
			opPar = i + 1
		case ']':
			opEnd = i
			fini = true
		case '\n', '\r', '/':
			fini = true
		default:
		}
		if fini {break}
	}

	if opEnd <= opPar {fmt.Errorf("no matching square brackets in seq!")}

//fmt.Printf("brack [%d: %d]: \"%s\"\n", opPar, opEnd, string(buf[opPar:opEnd]))

	arBuf := buf[opPar:opEnd]
	pgList, err := pdf.parseIrefArray(&arBuf)
	if err != nil {return fmt.Errorf("parseArray: %v", err)}

	pdf.pageCount = len(*pgList)
	pdf.pageIds = pgList

	fmt.Printf("Objects: %v Count: %d\n",pgList, len(*pgList))
	return nil
}


func (pdf *InfoPdf) parseIrefArray(inbuf *[]byte)(idList *[]int, err error) {

	var pg []int

	buf := *inbuf

	val := 0
	pgId := -1

	st := 0
	refCount := 0
	iref := -1

	for i:= 0; i< len(buf); i++ {
		if buf[i] == 'R' {
			iref = i
			refCount++
			_, errsc := fmt.Sscanf(string(buf[st:iref+1]),"%d %d R", &pgId, &val)
			if errsc != nil {return nil, fmt.Errorf("scan error obj %d: %v", refCount, errsc)}
			pg = append(pg, pgId)
			st = i+1
		}
	}
	return &pg, nil
}

//aa
func (pdf *InfoPdf) parseMbox(obj pdfObj)(mBox *[4]float32, err error) {

	var mbox [4]float32

	buf := *pdf.buf
	objByt := buf[obj.contSt:obj.contEnd]
//fmt.Printf("Mbox: %s\n",string(objByt))

	idx := bytes.Index(objByt, []byte("/MediaBox"))
	if idx == -1 {return nil, fmt.Errorf("cannot find keyword \"/MediaBox\"")}

	opPar := -1
	opEnd := -1

//fmt.Printf("brack start: %s\n", string(buf[(obj.contSt +5 + idx):obj.contEnd]))

	fini := false
	for i:= obj.contSt +idx + 5; i< obj.contEnd; i++ {
		switch buf[i] {
		case '[':
			opPar = i + 1
		case ']':
			opEnd = i
			fini = true
		case '\n', '\r', '/':
			fini = true
		default:
		}
		if fini {break}
	}

	if opEnd <= opPar {return nil, fmt.Errorf("no matching square brackets in seq!")}

//fmt.Printf("brack [%d: %d]: %s\n", opPar, opEnd, string(buf[opPar:opEnd]))

	// parse references
	_, errsc := fmt.Sscanf(string(buf[opPar:opEnd]),"%f %f %f %f", &mbox[0], &mbox[1], &mbox[2], &mbox[3])
	if errsc != nil {return nil, fmt.Errorf("scan error mbox: %v", errsc)}

//fmt.Printf("mbox: %v\n",mbox)
	return &mbox, nil
}


//eee
func (pdf *InfoPdf) findKeyWord(key string, obj pdfObj)(ipos int) {

	buf:= *pdf.buf
	objByt := buf[obj.contSt:obj.contEnd]
//fmt.Printf("find key: %s in %s\n", key, string(objByt))

	keyByt := []byte("/" + key)
	ipos = bytes.Index(objByt, keyByt)

	return ipos
}

func (pdf *InfoPdf) findVal(start, end int)(ipos int) {

	ipos = end
	buf:= *pdf.buf
	objByt := buf[start:end]

fmt.Printf("find val:%s \n", string(objByt))

	for i:=0; i< len(objByt); i++ {
		if objByt[i] == '/' {
			ipos = i
			break
		}
		if objByt[i] == '\n' || objByt[i] =='\r' {
			ipos = i
			break
		}
	}

fmt.Printf("find val end pos:%d \n", ipos)

	return ipos
}

func (pdf *InfoPdf) decodeObjStr(objId int)(outstr string, err error) {

	obj := (*pdf.objList)[objId]
	if obj.start == 0 {
		obj.typstr = "Invalid"
		(*pdf.objList)[objId] = obj
		return "invalid object", nil
	}

	buf := *pdf.buf


	valst := -1
	valend := -1
	ipos :=-1
	xend := -1
	obj.streamSt = -1
	obj.streamEnd = -1

	// exception for info
	if objId+1 == pdf.infoId {obj.typstr = "Info"}
	(*pdf.objList)[objId] = obj

	// no vals for contSt and contEnd
	_, obj.contSt, err = pdf.readLine(obj.start)
	if err != nil {return "", fmt.Errorf("no eol for obj %d: %v", objId, err)}

	// find endobj
	objByt := buf[obj.contSt:obj.end]
	xres := bytes.Index(objByt, []byte("endobj"))
	if xres == -1 {return "", fmt.Errorf("no endobj for obj %d!", objId)}

	endobj := obj.contSt + xres
	obj.contEnd = endobj

	// find stream
	objByt = buf[obj.contSt:endobj]
	xres = bytes.Index(objByt, []byte("stream"))
	if xres > -1 {
		obj.streamSt = obj.contSt + xres
		obj.contEnd = obj.streamSt
	}

	objByt = buf[obj.contSt:obj.contEnd]

//fmt.Printf("\nobj: %d stream: %d [%d:%d]: %s\n", objId, obj.streamSt, obj.contSt, obj.contEnd, string(objByt))

	obj.simple = false

	// is dictionary
	xres = bytes.Index(objByt, []byte("<<"))

	if xres <0 {
		obj.dict = false
		goto endParse
	}

	obj.dict = true

	xend = bytes.LastIndex(objByt, []byte(">>"))
	if xend == -1 {
		outstr += "no closing brackets in dict of obj"
		return outstr + "\n", fmt.Errorf(outstr)
	}

	obj.contEnd = obj.contSt + xend +2

//fmt.Printf("obj %d dict after (<<>>) [%d:%d]: %s\n", objId, obj.contSt, obj.contEnd, string(buf[obj.contSt:obj.contEnd]))

	// check type
	ipos = bytes.Index(objByt, []byte("/Type"))
	if ipos == -1 {goto endParse}

	// has type
	for i:= ipos+5; i< len(objByt); i++ {
		if objByt[i] == '/' {
			valst = i
			break
		}
	}
	if valst == -1 {return "", fmt.Errorf("no name for /type in obj %d found!", objId)}

//fmt.Printf("\nafter /type: %s valst %d: %s\n", string(objByt[ipos:ipos+5]), valst, string(objByt[valst:20]))

	for i:= valst+1; i< len(objByt); i++ {
		switch objByt[i] {
		case '/', '\r', '\n':
			valend = i
		default:
		}
		if valend> -1 {break}
	}
	if valend == -1 {return "", fmt.Errorf("no eol for val of /type in obj %d found!", objId)}

	obj.typstr = string(objByt[(valst +1):valend])
//fmt.Printf("obj: %d valstr [%d:%d]: %s\n", objId, valst, valend, obj.typstr)


	(*pdf.objList)[objId] = obj

	endParse:

	//getstream
	outstr = string(objByt)
	if obj.streamSt < 0 {
		outstr += "\nno keyword stream"
		return outstr + "\n", nil
	}

	outstr += "has stream\n"

	xres = bytes.Index(buf[obj.streamSt:obj.end], []byte("endstream"))

	if xres == -1 {
		outstr += " cannot find \"endstream\"\n"
		return outstr + "\n", fmt.Errorf(outstr)
	}

	obj.streamEnd = obj.streamSt + xres

	(*pdf.objList)[objId] = obj

	return outstr, nil
}

func (pdf *InfoPdf) PrintPdf() {

	fmt.Println("\n******************** Info Pdf **********************\n")
	fmt.Printf("File Name: %s\n", pdf.filNam)
	fmt.Printf("File Size: %d\n", pdf.filSize)
	fmt.Println()

	fmt.Printf("Page Count: %3d\n", pdf.pageCount)
	if pdf.mediabox == nil {
		fmt.Printf("no MediaBox\n")
	} else {
		fmt.Printf("MediaBox:    ")
		for i:=0; i< 4; i++ {
			fmt.Printf(" %.2f", (*pdf.mediabox)[i])
		}
	}
	if pdf.fonts == nil {
		fmt.Println("-- no Fonts")
	} else {
		fmt.Println("-- Font Ids:")
		for i:=0; i< len(*pdf.fonts); i++ {
				fmt.Printf("   %s %d\n", (*pdf.fonts)[i].Nam, (*pdf.fonts)[i].Id)
		}
	}

	if pdf.gStates == nil {
		fmt.Println("-- no ExtGstates")
	} else {
		fmt.Println("-- ExtGstate Ids:")
		for i:=0; i< len(*pdf.gStates); i++ {
			fmt.Printf("   %s %d\n", (*pdf.gStates)[i].Nam, (*pdf.gStates)[i].Id)
		}
	}

	if pdf.xObjs == nil {
		fmt.Println("-- no xObjs")
	} else {
		fmt.Println("-- xObj Ids:")
		for i:=0; i< len(*pdf.xObjs); i++ {
			fmt.Printf("   %s %d\n", (*pdf.xObjs)[i].Nam, (*pdf.xObjs)[i].Id)
		}
	}


	fmt.Println()
	fmt.Println()
	fmt.Printf("Info Id:    %5d\n", pdf.infoId)
	fmt.Printf("Root Id:    %5d\n", pdf.rootId)
	fmt.Printf("Pages Id:   %5d\n", pdf.pagesId)


	fmt.Printf("Xref Loc:      %5d\n", pdf.xref)
	fmt.Printf("trailer Loc:   %5d\n", pdf.trailer)
	fmt.Printf("startxref Loc: %5d\n", pdf.startxref)

	fmt.Println()
	fmt.Printf("*********************** xref Obj List [%3d] *********************\n", pdf.numObj)
	fmt.Printf("Objects: %5d    First Object Start Pos: %2d\n", pdf.numObj, pdf.objStart)
	fmt.Println("*****************************************************************")

	if pdf.objList == nil {
		fmt.Println("objlist is nil!")
		return
	}

	fmt.Println("                             Content      Stream")
	fmt.Println("Obj   Id type start  end   Start  End  Start  End  Length Type")
	for i:= 0; i< len(*pdf.objList); i++ {
		obj := (*pdf.objList)[i]
		fmt.Printf("%3d: %3d  %2d  %5d %5d %5d %5d %5d %5d %5d   %-15s\n",
		i, obj.objId, obj.typ, obj.start, obj.end, obj.contSt, obj.contEnd, obj.streamSt, obj.streamEnd, obj.streamEnd - obj.streamSt, obj.typstr)
	}
	fmt.Println()

	fmt.Println("*********** sequential Obj List *********************")
	if pdf.objList == nil {
		fmt.Println("rdObjlist is nil!")
		return
	}

	fmt.Println("Obj seq  Id start  type")
	for i:= 0; i< len(*pdf.rdObjList); i++ {
		obj := (*pdf.rdObjList)[i]
		fmt.Printf("%3d     %3d %5d\n", i, obj.objId, obj.start)
	}
	fmt.Println("************************************************")
	fmt.Println()

	for ipg:=0; ipg< pdf.pageCount; ipg++ {
		pdf.PrintPage(ipg)
	}
	return
}

func (pdf *InfoPdf) PrintPage (iPage int) {
		pgobj := (*pdf.pageList)[iPage]
		fmt.Printf("****************** Page %d *********************\n", iPage + 1)
		fmt.Printf("Page Number:     %d\n", pgobj.pageNum)
		fmt.Printf("Contents Obj Id: %d\n", pgobj.contentId)
		fmt.Printf("Media Box: ")
		if pgobj.mediabox == nil {
			fmt.Printf("no\n")
		} else {
			mbox := pgobj.mediabox
			for i:=0; i< 4; i++ {fmt.Printf(" %.1f", mbox[i])}
			fmt.Printf("\n")
		}
		fmt.Printf("Resources:\n")
		if pgobj.fonts == nil {
			fmt.Println("-- no Fonts")
		} else {
			fmt.Println("-- Font Ids:")
			for i:=0; i< len(*pgobj.fonts); i++ {
				fmt.Printf("   %s %d\n", (*pgobj.fonts)[i].Nam, (*pgobj.fonts)[i].Id)
			}
		}
		if pgobj.gStates == nil {
			fmt.Println("-- no ExtGstates")
		} else {
			fmt.Println("-- ExtGstate Ids:")
			for i:=0; i< len(*pgobj.gStates); i++ {
				fmt.Printf("   %s %d\n", (*pgobj.gStates)[i].Nam, (*pgobj.gStates)[i].Id)
			}
		}
		if pgobj.xObjs == nil {
			fmt.Println("-- no XObjects")
		} else {
			fmt.Println("-- XObject Ids:")
			for i:=0; i< len(*pgobj.xObjs); i++ {
				fmt.Printf("   %s %d\n", (*pgobj.xObjs)[i].Nam, (*pgobj.xObjs)[i].Id)
			}
		}
		fmt.Println("**********************************************")

}

//rr
func (pdf *InfoPdf) readLine(stPos int)(outstr string, nextPos int, err error) {

	buf := *pdf.buf

//fmt.Println("********")
	endPos := -1

	maxPos := stPos + 3000
	if len(buf) < maxPos {maxPos = len(buf)}
if pdf.test {fmt.Printf("\nreadLine [%d:%d]:\n%s\n***\n", stPos, maxPos, string(buf[stPos:maxPos]))}

	for i:=stPos; i < maxPos; i++ {
if pdf.test	{fmt.Printf("i: %d char: %q\n",i, buf[i])}
		if buf[i] == '\n' || buf[i] == '\r'{
			endPos = i
			nextPos = i+1
			if buf[i+1] == '\n' {nextPos++}

if pdf.test {fmt.Println("endpos: ", endPos)}
			break
		}
	}

	if endPos == -1 {return "", -1, fmt.Errorf("no eol found!")}

	outstr = string(buf[stPos:endPos])

if pdf.test {fmt.Printf("out:\n%s\nnext: %d\n", outstr, nextPos)}
	return outstr, nextPos, nil
}


func (pdf *InfoPdf) GetPdfObjList(buf *[]byte)(objList *[]pdfObj, err error) {

	fmt.Println("****************GetPdfObjList************")
	bufLen := len(*buf)
	fmt.Printf("GetPdfObjList: %d\n", bufLen)
	fmt.Println("pdf Objects: ", pdf.numObj)
	objList = pdf.objList
	for i:= 1; i< pdf.numObj; i++ {
		fmt.Printf("obj: %d start: %d end %d\n",i,(*objList)[i].start, (*objList)[i].end)
	}
	fmt.Println("****************GetPdfObjList************")

	return objList, err
}

func getObjTypStr(ityp int)(s string) {
	switch ityp {
		case 1:
			s = "Title"
		case 2:
			s = "Catalog"
		case 3:
			s = "Pages"
		case 4:
			s = "Page"
		case 5:
			s = "Font"
		case 6:
			s = "Font Desc"
		case 7:
			s = "Data"
		case 8:
			s= "ca"
		default:
			s= "unknown"
	}
	return s
}


func (pdf *InfoPdf) CreatePdfObj(pgNum int)(err error) {




//	pdf.objList = &pdfObjList
	return nil
}

func (pdf *InfoPdf) CreatePdf(pdfFilnam string)(err error) {

	var outstr string
	var pdfobj pdfObj
	var pdfObjList []pdfObj


	err = util.CheckFilnam(pdfFilnam, ".pdf")
	if err != nil {return fmt.Errorf("no pdf extension %s: %v\n", pdfFilnam, err);}

	pdfFil, err := os.Create(pdfFilnam)
	if err != nil {return fmt.Errorf("could not create pdf File %s: %v\n", pdfFilnam, err);}
	defer pdfFil.Close()

	// write top two lines
	tl := []byte("%pdf-1.4\n")
	sl := []byte{37,130, 131, 132, 10}

	bytSl := append(tl, sl...)
	pdfFil.Write(bytSl)

//	objSt := len(bytSl)


	// write objects

	var objSl []byte

	// info
	objStart, err := pdfFil.Seek(0, os.SEEK_END)
	if err != nil {return fmt.Errorf("seek %v", err)}
	objSt := int(objStart) + 1
	objBegin := int(objStart)
	objEnd := -1

	pdf.numObj++
	objStr := fmt.Sprintf("%d 0 obj\n", pdf.numObj)
	objLin := []byte(objStr)
	objSl = append(objSl, objLin...)

	objLin = []byte("<</Title (test)\n")
	objSl = append(objSl, objLin...)
	objLin = []byte("/Producer (azulPdf)>>\n")
	objSl = append(objSl, objLin...)

	objLin = []byte("endobj\n")
	objSl = append(objSl, objLin...)

	objEnd = objBegin + len(objSl)
//fmt.Printf("info: %d %d\n", objSt, objEnd)

	pdfobj.objId = pdf.numObj
	pdfobj.start = objSt
	pdfobj.end = objEnd
	pdfObjList = append(pdfObjList, pdfobj)
	pdf.infoId = pdf.numObj

	// root
	objSt = objEnd + 1
	objEnd = -1

	pdf.numObj++
	pdfobj.objId = pdf.numObj
	pdf.rootId = pdf.numObj
	objStr = fmt.Sprintf("%d 0 obj\n", pdf.numObj)
	objLin = []byte(objStr)

	objSl = append(objSl, objLin...)

	objLin = []byte("<</Type /Catalog\n")
	objSl = append(objSl, objLin...)

	objStr = fmt.Sprintf("/Pages %d 0 R>>\n", pdf.numObj +1)
	objLin = []byte(objStr)
//	objLin = []byte("/Pages 3 0 R>>\n")
	objSl = append(objSl, objLin...)

	objLin = []byte("endobj\n")
	objSl = append(objSl, objLin...)

//	pdfFil.Write(objSl)
	objEnd = objBegin + len(objSl)
//fmt.Printf("root: %d %d\n", objSt, objEnd)

	pdfobj.start = int(objSt)
	pdfobj.end = int(objEnd)
	pdfObjList = append(pdfObjList, pdfobj)

	// pages
	objSt = objEnd + 1
	objEnd = -1

	pdf.numObj++
	pdfobj.objId = pdf.numObj
	pdf.pagesId = pdf.numObj
	objStr = fmt.Sprintf("%d 0 obj\n", pdf.numObj)
	objLin = []byte(objStr)
//	objLin = []byte("3 0 obj\n")
	objSl = append(objSl, objLin...)

	objLin = []byte("<</Type /Pages\n")
	objSl = append(objSl, objLin...)

	// need to adjust for multiple pages

	objStr = fmt.Sprintf("/Kids [%d 0 R]\n", pdf.numObj +1)
	objLin = []byte(objStr)
//	objLin = []byte("/Kids [4 0 R]\n")
	objSl = append(objSl, objLin...)

	pgCount := 1

	objStr = fmt.Sprintf("/Count %d>>\n", pgCount)
	objLin = []byte(objStr)
//	objLin = []byte("/Count 1>>\n")
	objSl = append(objSl, objLin...)

	objLin = []byte("endobj\n")
	objSl = append(objSl, objLin...)

	objEnd = objBegin + len(objSl)

//fmt.Printf("objSt: %d %d\n", objSt, objEnd)

	pdfobj.start = int(objSt)
	pdfobj.end = int(objEnd)
	pdfObjList = append(pdfObjList, pdfobj)

	// page
	objSt = objEnd + 1
	objEnd = -1

	pdf.numObj++
	pdfobj.objId = pdf.numObj
	objStr = fmt.Sprintf("%d 0 obj\n", pdf.numObj)
	objLin = []byte(objStr)
//	objLin = []byte("4 0 obj\n")
	objSl = append(objSl, objLin...)

	objLin = []byte("<</Type /Page\n")
	objSl = append(objSl, objLin...)

	objStr = fmt.Sprintf("/Contents %d 0 R>>\n", pdf.numObj + 1)
	objLin = []byte(objStr)
//	objLin = []byte("/Contents 5 0 R>>\n")
	objSl = append(objSl, objLin...)

	objLin = []byte("endobj\n")
	objSl = append(objSl, objLin...)

	objEnd = objBegin + len(objSl)

//fmt.Printf("objSt: %d %d\n", objSt, objEnd)

	pdfobj.start = int(objSt)
	pdfobj.end = int(objEnd)
	pdfObjList = append(pdfObjList, pdfobj)

	// Content
	objSt = objEnd + 1
	objEnd = -1

	pdf.numObj++
	pdfobj.objId = pdf.numObj
	objStr = fmt.Sprintf("%d 0 obj\n", pdf.numObj)
	objLin = []byte(objStr)
//	objLin = []byte("5 0 obj\n")
	objSl = append(objSl, objLin...)

	objLin = []byte("<</Type /Contents\n")
	objSl = append(objSl, objLin...)

	streamLen := 99
	str := fmt.Sprintf("/Length %d\n", streamLen)
	objLin = []byte(str)
	objSl = append(objSl, objLin...)

	objLin = []byte("/Filter /FlateDecode>>\n")
	objSl = append(objSl, objLin...)


	objLin = []byte("stream\n")
	objSl = append(objSl, objLin...)
	// insert stream

	objLin = []byte("endstream\n")
	objSl = append(objSl, objLin...)


	objLin = []byte("endobj\n")
	objSl = append(objSl, objLin...)

	objEnd = objBegin + len(objSl)

//fmt.Printf("objSt: %d %d\n", objSt, objEnd)

	pdfobj.start = int(objSt)
	pdfobj.end = int(objEnd)
	pdfObjList = append(pdfObjList, pdfobj)


// Font
	objSt = objEnd + 1
	objEnd = -1

	pdf.numObj++
	pdfobj.objId = pdf.numObj
	objStr = fmt.Sprintf("%d 0 obj\n", pdf.numObj)
	objLin = []byte(objStr)
//	objLin = []byte("4 0 obj\n")
	objSl = append(objSl, objLin...)

	objLin = []byte("<</Type /Font\n")
	objSl = append(objSl, objLin...)
	objLin = []byte("/Subtype /TrueType\n")
	objSl = append(objSl, objLin...)
	objLin = []byte("/BaseFont /\"aaabbb + font\"\n")
	objSl = append(objSl, objLin...)
	objStr = fmt.Sprintf("/FontDescriptor %d 0 R>>\n", pdf.numObj + 1)
	objLin = []byte(objStr)


	objLin = []byte("endobj\n")
	objSl = append(objSl, objLin...)

	objEnd = objBegin + len(objSl)

//fmt.Printf("objSt: %d %d\n", objSt, objEnd)

	pdfobj.start = int(objSt)
	pdfobj.end = int(objEnd)
	pdfObjList = append(pdfObjList, pdfobj)

// Font Descriptor
	objSt = objEnd + 1
	objEnd = -1

	pdf.numObj++
	pdfobj.objId = pdf.numObj
	objStr = fmt.Sprintf("%d 0 obj\n", pdf.numObj)
	objLin = []byte(objStr)
	//	objLin = []byte("4 0 obj\n")
	objSl = append(objSl, objLin...)

	objLin = []byte("<</Type /FontDescriptor\n")
	objSl = append(objSl, objLin...)
	objLin = []byte("/FontName /\"aaabbb + font\"\n")
	objSl = append(objSl, objLin...)


	objStr = fmt.Sprintf("/FontFile %d 0 R>>\n", pdf.numObj + 1)
	objLin = []byte(objStr)


	objLin = []byte("endobj\n")
	objSl = append(objSl, objLin...)

	objEnd = objBegin + len(objSl)

//fmt.Printf("objSt: %d %d\n", objSt, objEnd)

	pdfobj.start = int(objSt)
	pdfobj.end = int(objEnd)
	pdfObjList = append(pdfObjList, pdfobj)

	// FontFile
	objSt = objEnd + 1
	objEnd = -1

	pdf.numObj++
	pdfobj.objId = pdf.numObj
	objStr = fmt.Sprintf("%d 0 obj\n", pdf.numObj)
	objLin = []byte(objStr)
	//	objLin = []byte("4 0 obj\n")
	objSl = append(objSl, objLin...)

	objLin = []byte("stream\n")
	objSl = append(objSl, objLin...)
	// insert stream

	objLin = []byte("endstream\n")
	objSl = append(objSl, objLin...)

	objLin = []byte("endobj\n")
	objSl = append(objSl, objLin...)

	objEnd = objBegin + len(objSl)

//fmt.Printf("objSt: %d %d\n", objSt, objEnd)

	pdfobj.start = int(objSt)
	pdfobj.end = int(objEnd)
	pdfObjList = append(pdfObjList, pdfobj)

	pdfFil.Write(objSl)

	// write xref
	xref, err := pdfFil.Seek(0, os.SEEK_END)
	if err != nil {return fmt.Errorf("xref pos %v", err)}
	pdf.xref = int(xref)

	pdfFil.WriteString("xref\n")
	outstr = fmt.Sprintf("0 %d\n", pdf.numObj + 1)
	pdfFil.WriteString(outstr)
	outstr = fmt.Sprintf("0000000000 65535 f \n")
	pdfFil.WriteString(outstr)

	objSl = []byte{}
	for i:=0; i<len(pdfObjList); i++ {
		objLin = []byte("0000000000 00000 n \n")
		str := fmt.Sprintf("%d",pdfObjList[i].start)
//fmt.Printf("obj %d: %s\n", i, str)
		pos := 10 -len(str)
		for j:=0; j<len(str); j++ {
			objLin[pos] = str[j]
			pos++
		}

		objSl = append(objSl, objLin...)
	}
	pdfFil.Write(objSl)

	// write trailer
	pdfFil.WriteString("trailer\n")
	outstr = fmt.Sprintf("<</Size %d\n", pdf.numObj)
	pdfFil.WriteString(outstr)
	outstr = fmt.Sprintf("/Root %d 0 R\n", pdf.rootId)
	pdfFil.WriteString(outstr)
	outstr = fmt.Sprintf("/Info %d 0 R>>\n", pdf.infoId)
	pdfFil.WriteString(outstr)

	// write last three lines
	pdfFil.WriteString("startxref\n")
	outstr = fmt.Sprintf("%d\n", pdf.xref)
	pdfFil.WriteString(outstr)
	outstr = "%%EOF"
	pdfFil.WriteString(outstr)
	return nil
}

