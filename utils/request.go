package utils

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/render"
	"github.com/pritunl/mongo-go-driver/bson/primitive"
)

var (
	privateCidrs    = []*net.IPNet{}
	privateCidrsStr = []string{
		"10.0.0.0/8",
		"100.64.0.0/10",
		"127.0.0.0/8",
		"172.16.0.0/12",
		"192.0.0.0/24",
		"192.168.0.0/16",
		"198.18.0.0/15",
		"6.0.0.0/8",
		"7.0.0.0/8",
		"11.0.0.0/8",
		"21.0.0.0/8",
		"22.0.0.0/8",
		"26.0.0.0/8",
		"28.0.0.0/8",
		"29.0.0.0/8",
		"30.0.0.0/8",
		"33.0.0.0/8",
		"::1/128",
		"fc00::/7",
		"fe80::/10",
	}
)

type NopCloser struct {
	io.Reader
}

func (NopCloser) Close() error {
	return nil
}

var httpErrCodes = map[int]string{
	400: "Bad Request",
	401: "Unauthorized",
	402: "Payment Required",
	403: "Forbidden",
	404: "Not Found",
	405: "Method Not Allowed",
	406: "Not Acceptable",
	407: "Proxy Authentication Required",
	408: "Request Timeout",
	409: "Conflict",
	410: "Gone",
	411: "Length Required",
	412: "Precondition Failed",
	413: "Payload Too Large",
	414: "URI Too Long",
	415: "Unsupported Media Type",
	416: "Range Not Satisfiable",
	417: "Expectation Failed",
	421: "Misdirected Request",
	422: "Unprocessable Entity",
	423: "Locked",
	424: "Failed Dependency",
	426: "Upgrade Required",
	428: "Precondition Required",
	429: "Too Many Requests",
	431: "Request Header Fields Too Large",
	451: "Unavailable For Legal Reasons",
	500: "Internal Server Error",
	501: "Not Implemented",
	502: "Bad Gateway",
	503: "Service Unavailable",
	504: "Gateway Timeout",
	505: "HTTP Version Not Supported",
	506: "Variant Also Negotiates",
	507: "Insufficient Storage",
	508: "Loop Detected",
	510: "Not Extended",
	511: "Network Authentication Required",
}

func StripPort(hostport string) string {
	colon := strings.IndexByte(hostport, ':')
	if colon == -1 {
		return hostport
	}
	if i := strings.IndexByte(hostport, ']'); i != -1 {
		return strings.TrimPrefix(hostport[:i], "[")
	}
	return hostport[:colon]
}

func FormatHostPort(hostname string, port int) string {
	if strings.Contains(hostname, ":") {
		hostname = "[" + hostname + "]"
	}
	return fmt.Sprintf("%s:%d", hostname, port)
}

func ParseObjectId(strId string) (objId primitive.ObjectID, ok bool) {
	objectId, err := primitive.ObjectIDFromHex(strId)
	if err != nil {
		return
	}

	objId = objectId
	ok = true
	return
}

func GetStatusMessage(code int) string {
	return fmt.Sprintf("%d %s", code, http.StatusText(code))
}

func AbortWithStatus(c *gin.Context, code int) {
	r := render.String{
		Format: GetStatusMessage(code),
	}

	c.Status(code)
	r.WriteContentType(c.Writer)
	c.Writer.WriteHeaderNow()
	r.Render(c.Writer)
	c.Abort()
}

func AbortWithError(c *gin.Context, code int, err error) {
	AbortWithStatus(c, code)
	c.Error(err)
}

func WriteStatus(w http.ResponseWriter, code int) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(code)
	fmt.Fprintln(w, GetStatusMessage(code))
}

func WriteText(w http.ResponseWriter, code int, text string) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(code)
	fmt.Fprintln(w, text)
}

func WriteUnauthorized(w http.ResponseWriter, msg string) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(401)
	fmt.Fprintln(w, "401 "+msg)
}

func CloneHeader(src http.Header) (dst http.Header) {
	dst = make(http.Header, len(src))
	for k, vv := range src {
		vv2 := make([]string, len(vv))
		copy(vv2, vv)
		dst[k] = vv2
	}
	return dst
}

func GetLocation(r *http.Request) string {
	host := ""

	switch {
	case r.Header.Get("X-Host") != "":
		host = r.Header.Get("X-Host")
		break
	case r.Host != "":
		host = r.Host
		break
	case r.URL.Host != "":
		host = r.URL.Host
		break
	}

	return "https://" + host
}

func ProxyUrl(srcUrl *url.URL, dstScheme, dstHost string) (
	dstUrl *url.URL) {

	dstUrl = &url.URL{
		Scheme:   dstScheme,
		Host:     dstHost,
		Path:     srcUrl.Path,
		Fragment: srcUrl.Fragment,
	}

	srcQuery := srcUrl.Query()
	dstQuery := url.Values{}

	for key, vals := range srcQuery {
		for _, val := range vals {
			dstQuery.Add(key, val)
		}
	}

	dstUrl.RawQuery = dstQuery.Encode()

	return
}

func CopyHeaders(dst, src http.Header) {
	for k, vv := range src {
		for _, v := range vv {
			dst.Add(k, v)
		}
	}
}

func IsPrivateRequest(r *http.Request) (private bool) {
	addr := net.ParseIP(StripPort(r.RemoteAddr))
	if addr == nil {
		return
	}

	for _, block := range privateCidrs {
		if block.Contains(addr) {
			private = true
			return
		}
	}

	return
}

func init() {
	for _, cidr := range privateCidrsStr {
		_, block, err := net.ParseCIDR(cidr)
		if err != nil {
			panic("Invalid private cidr")
		}
		privateCidrs = append(privateCidrs, block)
	}
}
