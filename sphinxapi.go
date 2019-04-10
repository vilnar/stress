package main

import (
	"encoding/binary"
	"errors"
	"fmt"
	"net"
)

// fixme! move whole this things into separate package for API
const (
	SPHINX_CLIENT_VERSION uint32 = 1
	SPHINX_SEARCHD_PROTO  uint32 = 1
)

const JSON_EOF byte = 0

/// known status return codes
type SearchdStatus_e uint16

const (
	SEARCHD_OK      = SearchdStatus_e(iota) ///< general success, command-specific reply follows
	SEARCHD_ERROR                           ///< general failure, error message follows
	SEARCHD_RETRY                           ///< temporary failure, error message follows, client should retry later
	SEARCHD_WARNING                         ///< general success, warning message and command-specific reply follow
)

/// known commands
type SearchdCommand_e uint16

const (
	SEARCHD_COMMAND_SEARCH = SearchdCommand_e(iota)
	SEARCHD_COMMAND_EXCERPT
	SEARCHD_COMMAND_UPDATE
	SEARCHD_COMMAND_KEYWORDS
	SEARCHD_COMMAND_PERSIST
	SEARCHD_COMMAND_STATUS
	_
	SEARCHD_COMMAND_FLUSHATTRS
	SEARCHD_COMMAND_SPHINXQL
	SEARCHD_COMMAND_PING
	SEARCHD_COMMAND_DELETE
	SEARCHD_COMMAND_UVAR
	SEARCHD_COMMAND_INSERT
	SEARCHD_COMMAND_REPLACE
	SEARCHD_COMMAND_COMMIT
	SEARCHD_COMMAND_SUGGEST
	SEARCHD_COMMAND_JSON
	SEARCHD_COMMAND_CALLPQ
	SEARCHD_COMMAND_CLUSTERPQ

	SEARCHD_COMMAND_TOTAL
	SEARCHD_COMMAND_WRONG = SEARCHD_COMMAND_TOTAL
)

/// known command versions
type SearchdCommandV_e uint16

const (
	VER_COMMAND_SEARCH     SearchdCommandV_e = 0x121
	VER_COMMAND_EXCERPT    SearchdCommandV_e = 0x104
	VER_COMMAND_UPDATE     SearchdCommandV_e = 0x103
	VER_COMMAND_KEYWORDS   SearchdCommandV_e = 0x101
	VER_COMMAND_STATUS     SearchdCommandV_e = 0x101
	VER_COMMAND_FLUSHATTRS SearchdCommandV_e = 0x100
	VER_COMMAND_SPHINXQL   SearchdCommandV_e = 0x100
	VER_COMMAND_JSON       SearchdCommandV_e = 0x100
	VER_COMMAND_PING       SearchdCommandV_e = 0x100
	VER_COMMAND_UVAR       SearchdCommandV_e = 0x100
	VER_COMMAND_CALLPQ     SearchdCommandV_e = 0x100
	VER_COMMAND_CLUSTERPQ  SearchdCommandV_e = 0x100

	VER_COMMAND_WRONG SearchdCommandV_e = 0
)

const VER_MASTER uint32 = 16

/// search query sorting orders
type ESphSortOrder uint32

const (
	SPH_SORT_RELEVANCE     = ESphSortOrder(iota) ///< sort by document relevance desc, then by date
	SPH_SORT_ATTR_DESC                           ///< sort by document date desc, then by relevance desc
	SPH_SORT_ATTR_ASC                            ///< sort by document date asc, then by relevance desc
	SPH_SORT_TIME_SEGMENTS                       ///< sort by time segments (hour/day/week/etc) desc, then by relevance desc
	SPH_SORT_EXTENDED                            ///< sort by SQL-like expression (eg. "@relevance DESC, price ASC, @id DESC")
	SPH_SORT_EXPR                                ///< sort by arithmetic expression in descending order (eg. "@id + max(@weight,1000)*boost + log(price)")
	SPH_SORT_TOTAL
)

/// search query matching mode
type ESphMatchMode uint32

const (
	SPH_MATCH_ALL       = ESphMatchMode(iota) ///< match all query words
	SPH_MATCH_ANY                             ///< match any query word
	SPH_MATCH_PHRASE                          ///< match this exact phrase
	SPH_MATCH_BOOLEAN                         ///< match this boolean query
	SPH_MATCH_EXTENDED                        ///< match this extended query
	SPH_MATCH_FULLSCAN                        ///< match all document IDs w/o fulltext query, apply filters
	SPH_MATCH_EXTENDED2                       ///< extended engine V2 (TEMPORARY, WILL BE REMOVED IN 0.9.8-RELEASE)

	SPH_MATCH_TOTAL
)

/// search query relevance ranking mode
type ESphRankMode uint32

const (
	SPH_RANK_PROXIMITY_BM25 = ESphRankMode(iota) ///< default mode, phrase proximity major factor and BM25 minor one (aka SPH03)
	SPH_RANK_BM25                                ///< statistical mode, BM25 ranking only (faster but worse quality)
	SPH_RANK_NONE                                ///< no ranking, all matches get a weight of 1
	SPH_RANK_WORDCOUNT                           ///< simple word-count weighting, rank is a weighted sum of per-field keyword occurence counts
	SPH_RANK_PROXIMITY                           ///< phrase proximity (aka SPH01)
	SPH_RANK_MATCHANY                            ///< emulate old match-any weighting (aka SPH02)
	SPH_RANK_FIELDMASK                           ///< sets bits where there were matches
	SPH_RANK_SPH04                               ///< codename SPH04, phrase proximity + bm25 + head/exact boost
	SPH_RANK_EXPR                                ///< rank by user expression (eg. "sum(lcs*user_weight)*1000+bm25")
	SPH_RANK_EXPORT                              ///< rank by BM25, but compute and export all user expression factors
	SPH_RANK_PLUGIN                              ///< user-defined ranker
	SPH_RANK_TOTAL
	SPH_RANK_DEFAULT = SPH_RANK_PROXIMITY_BM25
)

/// search query grouping mode
type ESphGroupBy int

const (
	SPH_GROUPBY_DAY      = iota ///< group by day
	SPH_GROUPBY_WEEK            ///< group by week
	SPH_GROUPBY_MONTH           ///< group by month
	SPH_GROUPBY_YEAR            ///< group by year
	SPH_GROUPBY_ATTR            ///< group by attribute value
	SPH_GROUPBY_ATTRPAIR        ///< group by sequential attrs pair (rendered redundant by 64bit attrs support; removed)
	SPH_GROUPBY_MULTIPLE        ///< group by on multiple attribute values
)

/// known collations
type ESphCollation uint32

const (
	SPH_COLLATION_LIBC_CI = ESphCollation(iota)
	SPH_COLLATION_LIBC_CS
	SPH_COLLATION_UTF8_GENERAL_CI
	SPH_COLLATION_BINARY

	SPH_COLLATION_DEFAULT = SPH_COLLATION_LIBC_CI
)

/// aggregate function to apply
type ESphAggrFunc uint32

const (
	SPH_AGGR_NONE = ESphAggrFunc(iota)
	SPH_AGGR_AVG
	SPH_AGGR_MIN
	SPH_AGGR_MAX
	SPH_AGGR_SUM
	SPH_AGGR_CAT
)

type QueryOption_e uint32

const (
	QUERY_OPT_DEFAULT = QueryOption_e(iota)
	QUERY_OPT_DISABLED
	QUERY_OPT_ENABLED
)

/// known attribute types
type ESphAttr uint32

const (
	// these types are full types
	// their typecodes are saved in the index schema, and thus,
	// TYPECODES MUST NOT CHANGE ONCE INTRODUCED
	SPH_ATTR_NONE       = ESphAttr(iota) ///< not an attribute at all
	SPH_ATTR_INTEGER                     ///< unsigned 32-bit integer
	SPH_ATTR_TIMESTAMP                   ///< this attr is a timestamp
	_                                    // there was SPH_ATTR_ORDINAL=3 once
	SPH_ATTR_BOOL                        ///< this attr is a boolean bit field
	SPH_ATTR_FLOAT                       ///< floating point number (IEEE 32-bit)
	SPH_ATTR_BIGINT                      ///< signed 64-bit integer
	SPH_ATTR_STRING                      ///< string (binary; in-memory)
	_                                    // there was SPH_ATTR_WORDCOUNT=8 once
	SPH_ATTR_POLY2D                      ///< vector of floats, 2D polygon (see POLY2D)
	SPH_ATTR_STRINGPTR                   ///< string (binary, in-memory, stored as pointer to the zero-terminated string)
	SPH_ATTR_TOKENCOUNT                  ///< field token count, 32-bit integer
	SPH_ATTR_JSON                        ///< JSON subset; converted, packed, and stored as string

	SPH_ATTR_UINT32SET = ESphAttr(0x40000001) ///< MVA, set of unsigned 32-bit integers
	SPH_ATTR_INT64SET  = ESphAttr(0x40000002) ///< MVA, set of signed 64-bit integers

	// these types are runtime only
	// used as intermediate types in the expression engine
	SPH_ATTR_MAPARG       = ESphAttr(1000 + iota)
	SPH_ATTR_FACTORS      ///< packed search factors (binary, in-memory, pooled)
	SPH_ATTR_JSON_FIELD   ///< points to particular field in JSON column subset
	SPH_ATTR_FACTORS_JSON ///< packed search factors (binary, in-memory, pooled, provided to client json encoded)

	SPH_ATTR_UINT32SET_PTR  // in-memory version of MVA32
	SPH_ATTR_INT64SET_PTR   // in-memory version of MVA64
	SPH_ATTR_JSON_PTR       // in-memory version of JSON
	SPH_ATTR_JSON_FIELD_PTR // in-memory version of JSON_FIELD
)

type docid uint64

const DOCID_MAX docid = 0xffffffffffffffff

type APIBuf []byte

func (buf *APIBuf) SendUint(val uint32) {
	tmp := make([]byte, 4)
	binary.BigEndian.PutUint32(tmp, val)
	*buf = append(*buf, tmp...)
}

func (buf *APIBuf) GetUint() uint32 {
	val := binary.BigEndian.Uint32(*buf)
	*buf = (*buf)[4:]
	return val
}

func (buf *APIBuf) SendUint64(val uint64) {
	tmp := make([]byte, 8)
	binary.BigEndian.PutUint64(tmp, val)
	*buf = append(*buf, tmp...)
}

func (buf *APIBuf) GetUint64() uint64 {
	val := binary.BigEndian.Uint64(*buf)
	*buf = (*buf)[8:]
	return val
}

func (buf *APIBuf) GetByte() byte {
	val := (*buf)[0]
	*buf = (*buf)[1:]
	return val
}

func (buf *APIBuf) SendDocid(val docid) {
	buf.SendUint64(uint64(val))
}

func (buf *APIBuf) SendInt(val int) {
	buf.SendUint(uint32(val))
}

func (buf *APIBuf) GetInt() int {
	return int(buf.GetUint())
}

func (buf *APIBuf) SendDword(val uint32) {
	buf.SendUint(val)
}

func (buf *APIBuf) GetDword() uint32 {
	return buf.GetUint()
}

func (buf *APIBuf) SendBytes(val *[]byte) {
	*buf = append(*buf, *val...)
}

func (buf *APIBuf) SendString(str string) {
	buf.SendInt(len(str))
	bytes := []byte(str)
	buf.SendBytes(&bytes)
}

func (buf *APIBuf) GetString() string {
	lng := buf.GetInt()
	result := string((*buf)[:lng])
	*buf = (*buf)[lng:]
	return result
}

func (buf *APIBuf) SendWord(val uint16) {
	tmp := make([]byte, 2)
	binary.BigEndian.PutUint16(tmp, val)
	*buf = append(*buf, tmp...)
}

func (buf *APIBuf) GetWord() uint16 {
	val := binary.BigEndian.Uint16(*buf)
	*buf = (*buf)[2:]
	return val
}

func (buf *APIBuf) Bytes() []byte {
	return []byte(*buf)
}

type SphinxClient struct {
	outb, backin APIBuf
	conn         net.Conn
	connected    bool
}

func (buf *APIBuf) APICommand(uCommand SearchdCommand_e, uVerOpt ...SearchdCommandV_e) int {
	uVer := uint16(0)
	if len(uVerOpt) > 0 {
		uVer = uint16(uVerOpt[0])
	}

	buf.SendWord(uint16(uCommand))
	buf.SendWord(uVer)

	iPlace := len(*buf)
	buf.SendUint(0) // space for future len encoding
	return iPlace
}

func (buf *APIBuf) FinishAPIPacket(iPlace int) {
	uLen := uint32(len(*buf) - iPlace - 4)
	binary.BigEndian.PutUint32((*buf)[iPlace:], uLen)
}

func (buf *APIBuf) buildSearchRequest(idx, query string, maxmatches int) {

	// fixme! cheat path - uflags must be calculated searchd.cpp 3140
	buf.SendDword(64) // uflags, means QFLAG_NORMALIZED_TF

	// The Search Legacy
	buf.SendInt(0) // offset is 0

	buf.SendInt(maxmatches) // limit

	buf.SendDword(uint32(SPH_MATCH_EXTENDED2))     // match mode
	buf.SendDword(uint32(SPH_RANK_PROXIMITY_BM25)) // ranking mode
	// if SPH_RANK_EXPR || SPH_RANK_EXPORT
	// buf.SendString (sRankerExpr)

	buf.SendInt(int(SPH_SORT_EXTENDED)) // sort mode
	buf.SendString("@weight desc")      // sort attr

	buf.SendString(query) // rawquery

	buf.SendInt(0) // num of weights
	// here zero of weights

	buf.SendString(idx) // indexes
	buf.SendInt(1)      // id range bits
	buf.SendDocid(0)    // default full id range (any client range must be in filters at this stage)
	buf.SendDocid(DOCID_MAX)

	buf.SendInt(0) // N of filters, zero for now
	// filters goes here

	buf.SendInt(SPH_GROUPBY_ATTR)
	buf.SendString("")      // groupby str, is empty for now
	buf.SendInt(maxmatches) // divided chunk

	buf.SendString("@groupby desc") // group by clause
	buf.SendInt(0)                  // cutoff

	buf.SendInt(0) // retrycount
	buf.SendInt(0) // retrydelay

	buf.SendString("") // group distinct

	buf.SendInt(0) // m_bGeoAnchor

	buf.SendInt(0) // num of index weights
	// index weights weights as pairs string index name, int value

	buf.SendDword(3000) // query timeout

	buf.SendInt(0) // num of fields weights
	// field weights as pairs string field name, int value

	buf.SendString("") // comment

	buf.SendInt(0) // N of overrides
	// overrides here

	buf.SendString("*") // select list
}

func (buf *APIBuf) buildSearchTail(mVer uint32) {
	// master-agent extensions

	// if uVer>=0x11B {
	// maxpredicted time m.b. send, but it is ruled by flags at the beginning
	//}

	// if uVer>=0x11D {
	// emulate empty sud-select for agent (client ver 1.29) as master sends fixed outer offset+limits
	buf.SendString("")
	buf.SendInt(0)
	buf.SendInt(0)
	buf.SendInt(0) // bHasOuter
	//}

	if mVer >= 1 {
		buf.SendDword(uint32(SPH_COLLATION_LIBC_CI))
	}

	if mVer >= 2 {
		buf.SendString("") // outer order by
		// if has outer SendInt outerlimit
	}

	if mVer >= 6 {
		buf.SendInt(1) // groupby limit
	}

	if mVer >= 14 {
		buf.SendString("") // m_sUDRanker
		buf.SendString("") // m_sUDRankerOpts
	}

	// if mVer>=14 || uVer>=0x120 {
	buf.SendString("") // m_sQueryTokenFilterLib
	buf.SendString("") // m_sQueryTokenFilterName
	buf.SendString("") // m_sQueryTokenFilterOpts
	// }

	// uVer>=0x121 {
	buf.SendInt(0) // N of filter tree elems
	// filter tree goes here
	//}

	if mVer >= 15 {

		buf.SendInt(1) // N of items
		// now the only item
		buf.SendString("*")
		buf.SendString("*")
		buf.SendDword(uint32(SPH_AGGR_NONE))

		buf.SendInt(0) // N of dRefItems
	}

	if mVer >= 16 {
		buf.SendDword(uint32(QUERY_OPT_DEFAULT)) // expand keyword
	}
}

func (cl *SphinxClient) getInBuf(size int) APIBuf {
	if len(cl.backin) < size {
		cl.backin = append(cl.backin, make([]byte, size-len(cl.backin))...)
	}
	return cl.backin[:size]
}

func (cl *SphinxClient) getOutBuf() *APIBuf {
	if cap(cl.outb) == 0 {
		cl.outb = APIBuf(make([]byte, 0, 256))
	} else {
		cl.outb = cl.outb[:0]
	}
	return &cl.outb
}

func (cl *SphinxClient) buildSearchClientRequest(idx, query string, maxmatches int) {
	buf := cl.getOutBuf()
	tPos := buf.APICommand(SEARCHD_COMMAND_SEARCH, VER_COMMAND_SEARCH)
	var mVer uint32 = 0 // this is client
	// outer layer
	buf.SendUint(mVer) // that is client!
	buf.SendUint(1)    // num of queries in batch
	buf.buildSearchRequest(idx, query, maxmatches)
	buf.buildSearchTail(mVer)
	buf.FinishAPIPacket(tPos)
}

func (cl *SphinxClient) buildSearchServerRequest(idx, query string, maxmatches int) {
	buf := cl.getOutBuf()
	tPos := buf.APICommand(SEARCHD_COMMAND_SEARCH, VER_COMMAND_SEARCH)
	var mVer = VER_MASTER // this is remote distr
	// outer layer
	buf.SendUint(mVer)
	buf.SendUint(1) // num of queries in batch
	buf.buildSearchRequest(idx, query, maxmatches)
	buf.buildSearchTail(mVer)
	buf.FinishAPIPacket(tPos)
}

func (cl *SphinxClient) buildHandshake() {
	handshake := cl.getOutBuf()
	handshake.SendUint(SPHINX_CLIENT_VERSION)
	pos := handshake.APICommand(SEARCHD_COMMAND_PERSIST)
	handshake.SendUint(1)
	handshake.FinishAPIPacket(pos)
}

func (cl *SphinxClient) Connect(uri string) (err error) {
	cl.conn, err = net.Dial("tcp", uri)
	if err == nil {
		cl.connected = true
	}
	return
}

func (cl *SphinxClient) Close() (err error) {
	err = cl.conn.Close()
	return
}

// establish sphinx API persistent connecton:
// sends initial SPHINX_CLIENT_VERSION, then command PERSIST
// receives handshake answer and check that it is correct
func (cl *SphinxClient) SendHandshake() (err error) {
	if !cl.connected {
		err = errors.New("client is NOT connected")
		return
	}
	cl.buildHandshake()
	_, err = cl.conn.Write(cl.outb)
	if err != nil {
		return
	}

	buf := cl.getInBuf(4)
	_, err = cl.conn.Read(buf)
	if err != nil {
		return
	}
	ver := buf.GetDword()
	if ver != SPHINX_SEARCHD_PROTO {
		err = errors.New(fmt.Sprintf("Wrong version num received: %d", ver))
	}
	return
}

// send Sphinx API simple request (only used fields are idx, query and maxmatches)
// receive and parse result, return num of rows returned from remote
func (cl *SphinxClient) SendServerSearch(idx, query string, maxmatches int) (count int, msg string, err error) {
	if !cl.connected {
		err = errors.New("client is NOT connected")
		return
	}
	count = 0
	cl.buildSearchServerRequest(idx, query, maxmatches)
	_, err = cl.conn.Write(cl.outb)
	if err != nil {
		return
	}

	rawrecv := cl.getInBuf(8)
	_, err = cl.conn.Read(rawrecv)
	if err != nil {
		return
	}
	uStat := SearchdStatus_e(rawrecv.GetWord())
	rawrecv.GetWord()
	iReplySize := rawrecv.GetInt()

	rawanswer := cl.getInBuf(iReplySize)
	_, err = cl.conn.Read(rawanswer)
	if err != nil {
		return
	}
	count, _, msg = parseSearchAnswer(&rawanswer, uStat)
	return
}

func (cl *SphinxClient) SendClientSearch(idx, query string, maxmatches int) (count int, msg string, err error) {
	if !cl.connected {
		err = errors.New("client is NOT connected")
		return
	}
	count = 0
	cl.buildSearchClientRequest(idx, query, maxmatches)
	_, err = cl.conn.Write(cl.outb)
	if err != nil {
		return
	}

	rawrecv := cl.getInBuf(8)
	_, err = cl.conn.Read(rawrecv)
	if err != nil {
		return
	}
	uStat := SearchdStatus_e(rawrecv.GetWord())
	rawrecv.GetWord()
	iReplySize := rawrecv.GetInt()

	rawanswer := cl.getInBuf(iReplySize)
	_, err = cl.conn.Read(rawanswer)
	if err != nil {
		return
	}
	count, _, msg = parseClientSendRequest(&rawanswer, uStat)
	return
}

type CSphIOStats struct {
	iReadTime, iReadBytes, iWriteTime, iWriteBytes int64
	iReadOps, iWriteOps                            uint32
}

type CSphQueryResult struct {
	sError, sWarning                            string
	iTotalMatches, iQueryTime, iReceivedMatches int
	CSphIOStats
	iCpuTime, iPredictedTime                                 int64
	iAgentFetchedDocs, iAgentFetchedHits, iAgentFetchedSkips uint32
	schema                                                   SimpleSchema
}

type ColumnInfo struct {
	sName     string
	eAttrType ESphAttr
}

type SimpleSchema []ColumnInfo

func (tRes *CSphQueryResult) ParseSchema(tReq *APIBuf) {
	nFields := tReq.GetInt()
	for j := 0; j < nFields; j++ {
		_ = tReq.GetString() // ignore fields
	}

	iNumAttrs := tReq.GetInt()
	tRes.schema = make([]ColumnInfo, iNumAttrs)
	for j := 0; j < iNumAttrs; j++ {
		tRes.schema[j].sName = tReq.GetString()
		tRes.schema[j].eAttrType = ESphAttr(tReq.GetDword())
	}
}

func (tRes *CSphQueryResult) ParseMatch(tReq *APIBuf, bAgent64 bool) {
	if bAgent64 {
		tReq.GetUint64()
	} else {
		tReq.GetDword()
	}
	tReq.GetInt() // iWeight
	for _, item := range tRes.schema {
		switch item.eAttrType {
		case SPH_ATTR_UINT32SET_PTR:
			iValues := int(tReq.GetDword())
			for i := 0; i < iValues; i++ {
				tReq.GetDword()
			}
		case SPH_ATTR_INT64SET_PTR:
			iValues := int(tReq.GetDword())
			for i := 0; i < iValues; i++ {
				tReq.GetUint64()
			}
		case SPH_ATTR_STRINGPTR:
		case SPH_ATTR_JSON_PTR:
		case SPH_ATTR_FACTORS:
		case SPH_ATTR_FACTORS_JSON:
			tReq.GetString()
		case SPH_ATTR_JSON_FIELD_PTR:
			jsontype := tReq.GetByte()
			if jsontype != JSON_EOF { // 0 is JSON_EOF
				tReq.GetString()
			}

		case SPH_ATTR_FLOAT:
			tReq.GetDword() // that is hack!

		case SPH_ATTR_BIGINT:
			tReq.GetUint64()

		default:
			tReq.GetDword()
		}
	}
}

func parseReplyHead(tReq *APIBuf) (tRes CSphQueryResult, err error) {
	eStatus := SearchdStatus_e(tReq.GetDword())
	switch eStatus {
	case SEARCHD_ERROR:
		tRes.sError = tReq.GetString()
		err = errors.New(tRes.sError)
		return
	case SEARCHD_RETRY:
		tRes.sError = tReq.GetString()
	case SEARCHD_WARNING:
		tRes.sWarning = tReq.GetString()
	}

	tRes.ParseSchema(tReq)

	tRes.iReceivedMatches = tReq.GetInt()
	bAgent64 := tReq.GetInt() != 0

	for i := 0; i < tRes.iReceivedMatches; i++ {
		tRes.ParseMatch(tReq, bAgent64)
	}

	// read totals (retrieved count, total count, query time, word count)
	_ = tReq.GetInt() // iRetrieved
	tRes.iTotalMatches = tReq.GetInt()
	tRes.iQueryTime = tReq.GetInt()
	return
}

func parseReply(tReq *APIBuf) (tRes CSphQueryResult) {
	var err error
	tRes, err = parseReplyHead(tReq)
	if err != nil {
		return
	}

	// agents always send IO/CPU stats to master
	uStatMask := tReq.GetByte()
	if uStatMask&1 != 0 {
		tRes.iReadTime = int64(tReq.GetUint64())
		tRes.iReadOps = tReq.GetDword()
		tRes.iReadBytes = int64(tReq.GetUint64())
		tRes.iWriteTime = int64(tReq.GetUint64())
		tRes.iWriteOps = tReq.GetDword()
		tRes.iWriteBytes = int64(tReq.GetUint64())
	}
	if uStatMask&2 != 0 {
		tRes.iCpuTime = int64(tReq.GetUint64())
	}

	if uStatMask&4 != 0 {
		tRes.iPredictedTime = int64(tReq.GetUint64())
	}

	tRes.iAgentFetchedDocs = tReq.GetDword()
	tRes.iAgentFetchedHits = tReq.GetDword()
	tRes.iAgentFetchedSkips = tReq.GetDword()

	iWordsCount := tReq.GetInt()

	// read per-word stats
	for i := 0; i < iWordsCount; i++ {
		_ = tReq.GetString() // word
		_ = tReq.GetInt()    // docs
		_ = tReq.GetInt()    // hits
		tReq.GetByte()       // statistics have no expanded terms for now
	}
	return
}

func parseClientReply(tReq *APIBuf) (tRes CSphQueryResult) {
	var err error
	tRes, err = parseReplyHead(tReq)
	if err != nil {
		return
	}
	iWordsCount := tReq.GetInt()
	// read per-word stats
	for i := 0; i < iWordsCount; i++ {
		_ = tReq.GetString() // word
		_ = tReq.GetInt()    // docs
		_ = tReq.GetInt()    // hits
	}
	return
}

func parseSearchAnswer(tReq *APIBuf, uStat SearchdStatus_e) (iMatches, iTime int, msg string) {
	switch uStat {
	case SEARCHD_ERROR:
		msg = tReq.GetString()
		return
	case SEARCHD_RETRY:
		msg = tReq.GetString()
		return
	case SEARCHD_WARNING:
		msg = tReq.GetString()
	}

	res := parseReply(tReq)
	iMatches = res.iReceivedMatches
	iTime = res.iQueryTime
	return
}

func parseClientSendRequest(tReq *APIBuf, uStat SearchdStatus_e) (iMatches, iTime int, msg string) {
	switch uStat {
	case SEARCHD_ERROR:
		msg = tReq.GetString()
		return
	case SEARCHD_RETRY:
		msg = tReq.GetString()
		return
	case SEARCHD_WARNING:
		msg = tReq.GetString()
	}

	res := parseClientReply(tReq)
	iMatches = res.iReceivedMatches
	iTime = res.iQueryTime
	return
}
