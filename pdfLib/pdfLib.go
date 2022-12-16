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
	fontIds *[]int
	fCount int
	gStateIds *[]int
	gCount int
	xObjIds *[]int
	xObjCount int
	pageList *[]pgObj
	objList *[]pdfObj
	rdObjList *[]pdfObj
	fontList *[]fontObj
	gStateList *[]gStateObj
	fonts *[]objRef
	gStates *[]objRef
	xObjs *[]objRef
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
	parent int
	start int
	end int
	contSt int
	contEnd int
	streamSt int
	streamEnd int
}

type pgObj struct {
	id int
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
}

type objRef struct {
	Id int
	Nam string
}

type fontObj struct {
	id int
	fontDesc *FontDesc
	fontDescId int
	subtyp string
	name string
	base string
	encode string
	desc int
	fchar int
	lchar int
	widths int
	widthList *[]int
}

type FontDesc struct {
	id int
	fname string
	flags int
	italic int
	ascent int
	descent int
	capheight int
	avgwidth	int
	maxwidth	int
	fontweight	int
	Xheight int
	fontBox [4]int
	fileId int
}

type gStateObj struct {
	BM string

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
	endPSl := endSl
	if buf[endPSl -1] == '\r' {endPSl--}


	startSl := endFl +1

	dif := endPSl - startSl

	if dif > 5 && pdf.verb {
		for i:=startSl; i< endPSl; i++ {
			fmt.Printf("[%d]:%d/%q ", i, buf[i], buf[i])
		}
	fmt.Printf("\n2 line [%d:%d]: %s\n", startSl, endSl, string(buf[startSl:endSl]))
	}

	if dif <5 || dif> 10 {return fmt.Errorf(" dif: %d no 4 chars after percent chars %d:%d!", dif, startSl, endSl)}

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

	// fix if buf[llEnd] == '\n\'
	if buf[llEnd] == '\n' {llEnd--}

	for i:=llEnd; i>llEnd -8; i-- {
		if buf[i] == '\n' {
			slEnd = i
			break
		}
	}

	if slEnd == -1 {return fmt.Errorf("cannot find eof for second top line")}

	idx := bytes.Index(buf[slEnd+1:], []byte("%%EOF"))
	if idx == -1 {return fmt.Errorf("last line %s: cannot find \"%%EOF\"!", string(buf[slEnd+1:]))}

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
			if endStr == "f" {pdfobj.start = -1}
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
			txtstr = fmt.Sprintf("getObjStr ObjId %d: %v", i, err)
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
// Step 10: parse each Font object
// Step 11: parse each related FontDescriptor object
// Step 12: parse each gstate object
// Step 13: parse each xobject object
//
	var outstr string

	txtFil, err := os.Create(txtfil)
	if err != nil {return fmt.Errorf("error creating textFile %s: %v\n", txtfil, err);}
	defer txtFil.Close()

	pdf.txtFil = txtFil

	buf := make([]byte,pdf.filSize)
	pdf.buf = &buf

	_, err = (pdf.fil).Read(buf)
	if err != nil {return fmt.Errorf("error Read: %v", err)}

	pdf.initDecode()

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

	// parse Trailer
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
	txtFil.WriteString("Obj ObjId  start  end   cont   stream  type\n")
	for i:=0; i< len(*rdObjList); i++ {
		obj := (*rdObjList)[i]
		outstr = fmt.Sprintf("%3d [%3d]: %5d %5d %5d %5d %s\n", i, obj.objId, obj.start, obj.end, obj.contSt, obj.contEnd, obj.typstr)
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
	txtFil.WriteString("Obj   Id type start  end   Start  End  Start  End  Length type\n")
	if pdf.verb {fmt.Printf("Obj ObjId  start  end   cont   stream  type\n")}
	for i:= 0; i< len(*pdf.objList); i++ {
		obj := (*pdf.objList)[i]
		outstr = fmt.Sprintf("%3d: %3d  %2d  %5d %5d %5d %5d %5d %5d %5d   %-15s\n",
		i, obj.objId, obj.typ, obj.start, obj.end, obj.contSt, obj.contEnd, obj.streamSt, obj.streamEnd, obj.streamEnd - obj.streamSt, obj.typstr)
		txtFil.WriteString(outstr)
		if pdf.verb {fmt.Printf(outstr)}
	}
	txtFil.WriteString("\n")

fmt.Println("\n*** parsing Pdf Tree ***\n")
	// create pdf dom tree


fmt.Printf("\n****** parsing Obj %d \"Catalog\" ******\n", pdf.rootId)
	err = pdf.parseRoot()
	if err != nil {return fmt.Errorf("parseRoot: %v", err)}
	txtFil.WriteString("parsed \"Root\" successfully!\n")
fmt.Printf("*** parsed Obj \"Catalog\" successfully ***\n")

fmt.Printf("\n******* parsing Obj %d \"Pages\" *******\n", pdf.pagesId)
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
		if err != nil {return fmt.Errorf("parsePageContent page %d: %v",ipg+1, err)}
		outstr = fmt.Sprintf("parsed Content successfully!")
		txtFil.WriteString(outstr +"\n")
		fmt.Printf("****** %s ********\n", outstr)

		fmt.Println()
	}

	// parse each Font object
	for i:=0; i< pdf.fCount; i++ {
		objId := (*pdf.fontIds)[i]
		fmt.Printf("*********** parsing Obj: %d Type: \"Font\" ************\n", objId)
		err = pdf.parseFont(objId)
		if err != nil {return fmt.Errorf("parseFont %d: %v",objId, err)}
		outstr := fmt.Sprintf("parsed \"Font %s ObjId: %d\" successfully!", "name", objId)
		txtFil.WriteString(outstr +"\n")
		fmt.Printf("***** parsed %s ******\n", outstr)
	}

	return nil
}

func (pdf *InfoPdf) initDecode() {
// method that initiates slices required in the decode process
	fontIds := make([]int, 10)
	pdf.fontIds = &fontIds

	gStateIds := make([]int, 10)
	pdf.gStateIds = &gStateIds

	return
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

	pgobj.pageNum = iPage + 1

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
//fmt.Printf("cont obj [%d:%d]:\n%s\n", obj.contSt, obj.contEnd, string(buf[obj.contSt:obj.contEnd]))

	txtFil := pdf.txtFil

	outstr := fmt.Sprintf("***** Content Page %d: id %d *******\n", iPage+1, contId)
	fmt.Printf(outstr)
	txtFil.WriteString(outstr)

	//Filter
	dictByt := buf[obj.contSt+2: obj.contEnd-2]
	key:="/Filter"

	filtStr, err := pdf.parseName(key, dictByt)
	if err != nil {return fmt.Errorf("parseName error parsing value of %s: %v",key, err)}
//fmt.Printf("key %s: val %s\n", key, filtStr)
	if filtStr != "FlateDecode" {
		outstr = fmt.Sprintf("Filter %s not implemented!\n", filtStr)
		fmt.Printf(outstr)
		txtFil.WriteString(outstr)
	}

	key = "/Length"
	streamLen, err := pdf.parseInt(key, dictByt)
	if err != nil {return fmt.Errorf("parseInt error: parsing value of %s: %v",key, err)}
//fmt.Printf("key %s: %d\n", key, streamLen)
	altStreamLen := obj.streamEnd - obj.streamSt
	if streamLen != altStreamLen {
		outstr = fmt.Sprintf("stream length inconsistent! Obj: %d Calc: %d\n", streamLen, altStreamLen)
		fmt.Printf(outstr)
		txtFil.WriteString(outstr)
	}

	// decode stream

	stream, err := pdf.decodeStream(contId)
	if err != nil {return fmt.Errorf("decodeStream: %v")}

fmt.Println("stream: ", string(*stream))

	txtFil.WriteString(string(*stream))

	return nil
}

func (pdf *InfoPdf) parseFont(objId int)(err error) {

	obj := (*pdf.objList)[objId]

	buf := *pdf.buf
fmt.Printf("font obj [%d:%d]:\n%s\n", obj.contSt, obj.contEnd, string(buf[obj.contSt:obj.contEnd]))

	txtFil := pdf.txtFil

	outstr := fmt.Sprintf("********* Font: id %d *******\n", objId)

	fmt.Printf(outstr)
	txtFil.WriteString(outstr)

	return nil
}

func (pdf *InfoPdf) parsePages()(err error) {

	if pdf.pagesId > pdf.numObj {return fmt.Errorf("invalid pagesId!")}
	if pdf.pagesId ==0 {return fmt.Errorf("pagesId is 0!")}

	obj := (*pdf.objList)[pdf.pagesId]

//fmt.Printf("pages:\n%s\n", string(buf[obj.start: obj.end]))

	err = pdf.parseKids(obj)
	if err!= nil {return fmt.Errorf("parseKids: %v", err)}

	if pdf.verb {
		fmt.Printf("pages: pageCount: %d\n", pdf.pageCount)
		for i:=0; i< pdf.pageCount; i++ {
			fmt.Printf("page: %d objId: %d\n", i+1, (*pdf.pageIds)[i])
		}
	}
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

	dictBytPt, err := pdf.parseDict("/Resources",objByt)
	if err != nil {return nil, fmt.Errorf("parseDict: %v", err)}
	dictByt := *dictBytPt
fmt.Printf("*** Resources dict ***\n%s\n", string(dictByt))
/*
	idx := bytes.Index(objByt, []byte("/Resources"))
	if idx == -1 {
		if pdf.verb {fmt.Printf("parseResource: cannot find keyword \"/Resources\"!\n")}
		return nil, nil
	}

	// either indirect or a dictionary
	valst := obj.contSt + idx + len("/Resources")
	objByt = buf[valst: obj.contEnd]
//fmt.Printf("Resources valstr [%d:%d]: %s\n",valst, obj.contEnd, string(objByt))
//ddd

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
*/
	// find Font
	if pdf.verb {fmt.Println("**** Font: dictionary *****")}
	fidx := bytes.Index(dictByt, []byte("/Font"))
	if fidx == -1 {
		if pdf.verb {fmt.Println("parseResources: no keyword \"/Font\"!")}
		reslist.fonts = nil
	} else {
//rrr
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


	pvalst := pidx + len("/ProcSet")
	pByt := dictByt[pvalst:]
//fmt.Printf("ProcSet valstr [%d:%d]: %s\n",pvalst, dictEnd, string(pByt))

	parrSt := bytes.Index(pByt, []byte("["))
//fmt.Printf("font dictSt: %d\n", fdictSt)

	parrEnd := bytes.Index(pByt, []byte("]"))
	if parrEnd == -1 {return nil, fmt.Errorf("no end bracket for ProcSet array!")}

	parrSt += pvalst +1
	parrEnd += pvalst
	parrByt := pByt[parrSt:parrEnd]
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
		if pdf.verb {fmt.Printf("%s: indirect obj\n", keyname)}
		fvalend := -1
		for i:= 0; i< len(valByt); i++ {
			if valByt[i] == 'R' {
				fvalend = i+1
				break
			}
		}
		if fvalend == -1 {return nil, -1, fmt.Errorf("cannot find R for indirect obj of \"/%s\"", keyname)}
		inObjStr := string(valByt[:fvalend])

//fmt.Printf("ind obj: %s\n", inObjStr)

		rev := 0
		_, err = fmt.Sscanf(inObjStr,"%d %d R", &objId, &rev)
		if err != nil{return nil, -1, fmt.Errorf("cannot parse %s as indirect obj of \"/%s\": %v", inObjStr, keyname, err)}

		if pdf.verb {fmt.Printf("%s indirect Obj Id: %d\n", keyname, objId)}

		objSl, err := pdf.getObjCont(objId)
		if err != nil{return nil, -1, fmt.Errorf("cannot get content of obj %d: %v", objId, err)}

		valByt = *objSl
		fdictSt = bytes.Index(valByt, []byte("<<"))

	}

fmt.Printf("%s: valstr [%d:]: %s\n",keyname, fdictSt, string(valByt))

	fdictEnd := bytes.Index(valByt, []byte(">>"))
	if fdictEnd == -1 {return nil, -1, fmt.Errorf("no end brackets for dict of %s!", keyname)}

	keyDictByt = valByt[fdictSt +2 :fdictEnd]


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
//fmt.Printf("Kids obj: %s\n",string(objByt))

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

//	fmt.Printf("Objects: %v Count: %d\n",pgList, len(*pgList))
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


func (pdf *InfoPdf) parseInt(keyword string, objByt []byte)(num int, err error) {

	var indObj pdfObj

	buf := *pdf.buf

	keyByt := []byte(keyword)
	idx := bytes.Index(objByt, keyByt)
	if idx == -1 {return -1, fmt.Errorf("cannot find keyword \"%s\"", string(keyByt))}

	opSt := -1
	opEnd := -1

	valst := idx+len(keyByt)
	valByt := objByt[valst:]

//fmt.Printf("valstr: %s\n", string(valByt))

	// whether indirect obj
	inObjId := parseIndObjRef(objByt[valst:])
	// todo make sure inObjIs is valid

	if inObjId > -1 {
		indObj = (*pdf.objList)[inObjId]
		valByt = buf[indObj.contSt: indObj.contEnd]
	}

//fmt.Printf("parse num obj valByt: %s\n", string(valByt))

	endByt := []byte{'\n', '\r', '/', ' '}

	istate := 0
	for i:= 0; i< len(valByt); i++ {
		switch istate {
		case 0:
			if util.IsNumeric(valByt[i]) {opSt = i;istate =1}

		case 1:
			if isEnding(valByt[i], endByt) {opEnd= i; istate =2}

		default:
		}
		if istate == 2 {break}
	}

	if istate == 0 {return -1, fmt.Errorf("no number found!")}
	if istate == 1 {opEnd = len(valByt)}

	valBuf := valByt[opSt:opEnd]

//fmt.Printf("key /%s val[%d: %d]: \"%s\"\n", keyword, opSt, opEnd, string(valBuf))

	_, err = fmt.Sscanf(string(valBuf), "%d", &num)
	if err != nil {return -1, fmt.Errorf("cannot parse num: %v", err)}

	return num, nil
}

func (pdf *InfoPdf) parseFloat(keyword string, objByt []byte)(fnum float32, err error) {

	var indObj pdfObj

	buf:= *pdf.buf
	keyByt := []byte(keyword)

	idx := bytes.Index(objByt, keyByt)
	if idx == -1 {return -1.0, fmt.Errorf("cannot find keyword \"%s\"", string(keyByt))}

	opSt := -1
	opEnd := -1
	valst := idx+len(keyByt)
	valByt := objByt[valst:]

//fmt.Printf("valstr: %s\n", string(valByt))

	// whether indirect obj
	inObjId := parseIndObjRef(objByt[valst:])
	// todo make sure inObjIs is valid

	if inObjId > -1 {
		indObj = (*pdf.objList)[inObjId]
		valByt = buf[indObj.contSt: indObj.contEnd]
	}

//fmt.Printf("parse float obj str: %s\n", string(valByt))

	endByt := []byte{'\n', '\r', '/', ' '}

	istate := 0
	for i:= 0; i< len(valByt); i++ {
		switch istate {
		case 0:
			if util.IsNumeric(valByt[i]) {opSt = i;istate =1}

		case 1:
			if isEnding(valByt[i], endByt) {opEnd= i; istate =2}

		default:
		}
		if istate == 2 {break}
	}

	if istate == 0 {return -1, fmt.Errorf("no number found!")}
	if istate == 1 {opEnd = len(valByt)}

	valBuf := valByt[opSt:opEnd]


//fmt.Printf("key %s val[%d: %d]: \"%s\"\n", keyword, opSt, opEnd, string(valBuf))

	_, err = fmt.Sscanf(string(valBuf), "%f", &fnum)
	if err != nil {return -1, fmt.Errorf("cannot parse fnum: %v", err)}

	return fnum, nil
}

func (pdf *InfoPdf) parseString(keyword string, objByt []byte)(outstr string, err error) {

	var indObj pdfObj
	buf := *pdf.buf

	keyByt := []byte(keyword)

	idx := bytes.Index(objByt, keyByt)
	if idx == -1 {return "", fmt.Errorf("cannot find keyword \"%s\"", string(keyByt))}

	opSt := -1
	opEnd := -1
	valst := idx+len(keyByt)
	valByt := objByt[valst:]

fmt.Printf("valstr: %s\n", string(valByt))

	// whether indirect obj
	inObjId := parseIndObjRef(objByt[valst:])
	// todo make sure inObjIs is valid

	if inObjId > -1 {
		indObj = (*pdf.objList)[inObjId]
		valByt = buf[indObj.contSt: indObj.contEnd]
	}

	endByt := []byte{'\n', '\r', '/'}

	istate := 0
	for i:= 0; i< len(valByt); i++ {
		switch istate {
		case 0:
			if valByt[i] == '(' {opSt = i;istate =1}

		case 1:
			if valByt[i] == ')' {opEnd = i;istate =2}

			if isEnding(valByt[i], endByt) {opEnd= i; istate =3}

		default:
		}
		if istate > 1 {break}
	}

	switch istate {
	case 0:
		return "", fmt.Errorf("no open '(' found!")
	case 1, 3:
		return "", fmt.Errorf("no open ')' found!")
	case 2:
		if opEnd -1 < opSt +1 {return "", fmt.Errorf("inverted string [%d:%d]",opSt+1, opEnd-1)}

	default:
		return "", fmt.Errorf("unknown istate %d!", istate)
	}

	valBuf := valByt[opSt+1:opEnd -1]

fmt.Printf("key /%s val[%d: %d]: \"%s\"\n", keyword, opSt, opEnd, string(valBuf))

	_, err = fmt.Sscanf(string(valBuf), "%s", &outstr)
	if err != nil {return "", fmt.Errorf("cannot parse string: %v", err)}

	return outstr, nil
}

func (pdf *InfoPdf) parseName(keyword string, objByt []byte)(outstr string, err error) {

//	var indObj pdfObj
//	buf := *pdf.buf

	keyByt := []byte(keyword)

	idx := bytes.Index(objByt, keyByt)
	if idx == -1 {return "", fmt.Errorf("cannot find keyword \"%s\"", string(keyByt))}

	opSt := -1
	opEnd := -1
	valst := idx+len(keyByt)
	valByt := objByt[valst:]

//fmt.Printf("valstr: %s\n", string(valByt))

	endByt := []byte{'\n', '\r', '/'}
	istate := 0
	for i:= 0; i< len(valByt); i++ {
		switch istate {
		case 0:
			if valByt[i] == '/' {opSt = i;istate =1}

		case 1:
			if isEnding(valByt[i], endByt) {opEnd= i; istate =2}

		default:
		}
		if istate > 1 {break}
	}

	switch istate {
	case 0:
		return "", fmt.Errorf("no open '/' found!")
	case 1:
		opEnd = len(valByt)
	case 2:
		if opEnd -1 < opSt +1 {return "", fmt.Errorf("inverted string [%d:%d]",opSt+1, opEnd-1)}

	default:
		return "", fmt.Errorf("unknown istate %d!", istate)
	}

	valBuf := valByt[opSt:opEnd]

//fmt.Printf("key %s val[%d: %d]: \"%s\"\n", keyword, opSt, opEnd, string(valBuf))

	_, err = fmt.Sscanf(string(valBuf), "/%s", &outstr)
	if err != nil {return "", fmt.Errorf("cannot parse string: %v", err)}

	return outstr, nil
}

func (pdf *InfoPdf) parseDict(keyword string, objByt []byte)(dict *[]byte, err error) {

	var nestSt, nestEnd [10]int

	keyByt := []byte(keyword)
	buf := *pdf.buf

	idx := bytes.Index(objByt, keyByt)
	if idx == -1 {return nil, fmt.Errorf("cannot find keyword \"%s\"", string(keyByt))}

	valst := idx+len(keyByt)
	valByt := objByt[valst:]

	objId:=0
	ref :=0
	_, err = fmt.Sscanf(string(objByt[valst: valst+10])," %d %d R", &objId, &ref)
	if err == nil {
fmt.Printf("%s found ind obj ref: %d\n", keyword, objId)
		obj := (*pdf.objList)[objId]

		objDictByt := buf[obj.contSt:obj.contEnd]
		return &objDictByt, nil
	}

	nestlev := -1
	for i:= idx + len(keyByt); i< len(valByt) -1; i++ {
		if valByt[i] == '<' && valByt[i+1] == '<' {
			nestlev++
			nestSt[nestlev] = i
			if nestlev > 10 {return nil, fmt.Errorf("nesting level exceeds 10!")}
		}
		if valByt[i] == '>' && valByt[i+1] == '>' {
			if nestlev < 0 {return nil, fmt.Errorf("nestlev is less than zero!")}
			nestlev--
			nestEnd[nestlev] = i
			break
		}
	}


	dictByt := valByt[nestSt[0]:nestEnd[0]]
	return &dictByt, nil
}

func parseIndObjRef(valByt []byte) (objId int) {
// function parses ValByt to find object reference
// if no obj id found return obj Id = -1
	valEnd := -1
	for i:=0; i< len(valByt); i++ {
		if valByt[i] == 'R' {
			valEnd = i
			break
		}
	}
	if valEnd == -1 {return -1}

	ref :=0
	_, err:= fmt.Sscanf(string(valByt[:valEnd+1]),"%d %d R", &objId, &ref)
	if err != nil {return -2}
	return objId
}

func isEnding (b byte, ending []byte)(end bool) {

	idx := -1
	for i:=0; i<len(ending); i++ {
		if ending[i] == b {
			idx = i
			break
		}
	}
	if idx == -1 {return false}
	return true
}

func (pdf *InfoPdf) findKeyWord(key string, obj pdfObj)(ipos int) {

	buf:= *pdf.buf
	objByt := buf[obj.contSt:obj.contEnd]
//fmt.Printf("find key: %s in %s\n", key, string(objByt))

	keyByt := []byte("/" + key)
	ipos = bytes.Index(objByt, keyByt)

	return ipos
}


func (pdf *InfoPdf) decodeObjStr(objId int)(outstr string, err error) {
// method parses an object to determine dict start/end stream start/end and object type

	buf := *pdf.buf

	valst := -1
	valend := -1
	ipos :=-1
	xend := -1

	err = pdf.parseObjHead(objId)
	if err != nil {return "", fmt.Errorf("parseObjHead: %v", err)}
//	if buf[obj.contSt] == '\r' {obj.contSt +=2} else {obj.contSt +=1}

	obj := (*pdf.objList)[objId]

	if obj.start < 0 {return "", nil}

	// exception for info
	if objId+1 == pdf.infoId {obj.typstr = "Info"}

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
		obj.contEnd = obj.contSt + xres
		obj.streamSt = obj.contSt + xres + 7
		if buf[obj.streamSt] == '\n' {obj.streamSt++}
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

// todo rep
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
		case '/', '\r', '\n', ' ':
			valend = i
		default:
		}
		if valend> -1 {break}
	}
	if valend == -1 {return "", fmt.Errorf("no eol for val of /type in obj %d found!", objId)}

	obj.typstr = string(objByt[(valst +1):valend])
//fmt.Printf("obj: %d valstr [%d:%d]: %s\n", objId, valst, valend, obj.typstr)
	switch obj.typstr {
	case "Font":
		(*pdf.fontIds)[pdf.fCount] = objId
		pdf.fCount++
	case "ExtGState":
		(*pdf.gStateIds)[pdf.gCount] = objId
		pdf.gCount++
	case "XObject":
		(*pdf.xObjIds)[pdf.xObjCount] = objId
		pdf.xObjCount++
	}

	endParse:

	(*pdf.objList)[objId] = obj

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

	obj.streamEnd = obj.streamSt + xres -1
	if buf[obj.streamEnd -1] == '\r' {obj.streamEnd--}

	(*pdf.objList)[objId] = obj
	return outstr, nil
}

func (pdf *InfoPdf) parseObjHead(objId int) (err error){


	buf := *pdf.buf

	if objId > pdf.numObj {return fmt.Errorf("invalid objId %d", objId)}
	obj := (*pdf.objList)[objId]

	obj.streamSt = -1
	obj.streamEnd = -1
	obj.contSt = -1
	obj.contEnd = -1
	if obj.start < 0 {
		(*pdf.objList)[objId]= obj
		return nil
	}

	linEnd := -1

	if obj.start == 0 {return fmt.Errorf("objId: %d obj.start value is invalid!", objId)}

	for i:= obj.start; i< obj.start + 10; i++ {
		if buf[i] == '\n' {
			linEnd = i
			break
		}
	}
	if linEnd == -1 {return fmt.Errorf("no eol found!")}

	obj.contSt = linEnd+1

	hdStr := string(buf[obj.start:linEnd])

	id:=0
	ref:=0
	_, err = fmt.Sscanf(hdStr,"%d %d obj", &id, &ref)
	if err != nil {return fmt.Errorf("obj %d: canno parse headline: %v!", err)}

	if id!= objId {return fmt.Errorf("objId %d does not match id %d in headline!", objId, id)}

	(*pdf.objList)[objId]= obj

	return nil
}

func (pdf *InfoPdf) PrintPdf() {

	fmt.Println("\n******************** Info Pdf **********************\n")
	fmt.Printf("File Name: %s\n", pdf.filNam)
	fmt.Printf("File Size: %d\n", pdf.filSize)
	fmt.Println()

	fmt.Printf("pdf version: major %d minor %d\n",pdf.majver, pdf.minver)

	fmt.Printf("Page Count: %3d\n", pdf.pageCount)
	if pdf.mediabox == nil {
		fmt.Printf("no MediaBox\n")
	} else {
		fmt.Printf("MediaBox:    ")
		for i:=0; i< 4; i++ {
			fmt.Printf(" %.2f", (*pdf.mediabox)[i])
		}
		fmt.Println()
	}
	fmt.Printf("Font Count: %d\n", pdf.fCount)
	fmt.Printf("Font Obj Ids:\n")
	for i:=0; i< pdf.fCount; i++ {
		fmt.Printf("%d\n", (*pdf.fontIds)[i])
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

	for i:=0; i< pdf.fCount; i++ {
		objId := (*pdf.fontIds)[i]
		fmt.Printf("*************** Font Obj %d ******************\n", objId)

	}
/*
	if pdf.fonts == nil {
		fmt.Println("-- no Fonts")
	} else {
		fmt.Println("-- Font Ids:")
		for i:=0; i< len(*pdf.fonts); i++ {
				fmt.Printf("   %s %d\n", (*pdf.fonts)[i].Nam, (*pdf.fonts)[i].Id)
		}
	}
*/
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

