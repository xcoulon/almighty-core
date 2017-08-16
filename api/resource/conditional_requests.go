package resource

import (
	"bytes"
	"crypto/md5"
	"encoding/base64"
	"fmt"
	"net/http"
	"reflect"
	"strconv"
	"time"

	"github.com/fabric8-services/fabric8-wit/log"
	workitemdsl "github.com/fabric8-services/fabric8-wit/workitem"
	uuid "github.com/satori/go.uuid"
)

const (
	// IfModifiedSince the "If-Modified-Since" HTTP request header name
	IfModifiedSince = "If-Modified-Since"
	// LastModified the "Last-Modified" HTTP response header name
	LastModified = "Last-Modified"
	// IfNoneMatch the "If-None-Match" HTTP request header name
	IfNoneMatch = "If-None-Match"
	// ETag the "ETag" HTTP response header name
	// should be ETag but GOA will convert it to "Etag" when setting the header.
	// Plus, RFC 2616 specifies that header names are case insensitive:
	// https://www.w3.org/Protocols/rfc2616/rfc2616-sec4.html#sec4.2
	ETag = "Etag"
	// CacheControl the "Cache-Control" HTTP response header name
	CacheControl = "Cache-Control"
	// MaxAge the "max-age" HTTP response header value
	MaxAge = "max-age"
)

type CacheControlConfig func() string

// ConditionalRequestContext interface with methods for the contexts
type ConditionalRequestContext interface {
	NotModified() error
	getIfModifiedSince() *time.Time
	setLastModified(time.Time)
	getIfNoneMatch() *string
	setETag(string)
	setCacheControl(string)
}

// ConditionalRequestEntity interface with methods for the response entities
type ConditionalRequestEntity interface {
	// returns the time of last update
	GetLastModified() time.Time
	// returns the values to use to generate the ETag
	GetETagData() []interface{}
}

func doConditionalRequest(ctx ConditionalRequestContext, entity ConditionalRequestEntity, cacheControlConfig CacheControlConfig, nonConditionalCallback func()) {
	lastModified := entity.GetLastModified()
	eTag := GenerateEntityTag(entity)
	cacheControl := cacheControlConfig()
	ctx.setLastModified(lastModified)
	ctx.setETag(eTag)
	ctx.setCacheControl(cacheControl)
	// check the 'If-None-Match' header first.
	found, match := matchesETag(ctx, eTag)
	if found && match {
		ctx.NotModified()
	} else if !found {
		// check the 'If-Modified-Since' header only if no 'If-None-Match' header was provided
		found, modified := modifiedSince(ctx, lastModified)
		if found && !modified {
			ctx.NotModified()
		}
	}
	// call the 'nonConditionalCallback' if the entity was modified since the client's last call
	nonConditionalCallback()
}

func doConditionalEntities(ctx ConditionalRequestContext, entities []ConditionalRequestEntity, cacheControlConfig CacheControlConfig, nonConditionalCallback func()) {
	var lastModified time.Time
	var eTag string
	if len(entities) > 0 {
		for _, entity := range entities {
			if entity.GetLastModified().After(lastModified) {
				lastModified = entity.GetLastModified()
			}
		}
		eTag = GenerateEntitiesTag(entities)
	} else {
		eTag = GenerateEmptyTag()
		lastModified = time.Now()
	}
	ctx.setLastModified(lastModified)
	ctx.setETag(eTag)
	cacheControl := cacheControlConfig()
	ctx.setCacheControl(cacheControl)
	// check the 'If-None-Match' header first.
	found, match := matchesETag(ctx, eTag)
	if found && match {
		ctx.NotModified()
	} else if !found {
		// check the 'If-Modified-Since' header only if no 'If-None-Match' header was provided
		found, modified := modifiedSince(ctx, lastModified)
		if found && !modified {
			ctx.NotModified()
		}
	}
	// call the 'nonConditionalCallback' if the entity was modified since the client's last call
	nonConditionalCallback()
}

// GenerateEmptyTag generates the value to return in the "ETag" HTTP response header for the an empty list of entities
// The ETag is the base64-encoded value of the md5 hash of the buffer content
func GenerateEmptyTag() string {
	var buffer bytes.Buffer
	buffer.WriteString("empty")
	etagData := md5.Sum(buffer.Bytes())
	etag := base64.StdEncoding.EncodeToString(etagData[:])
	return etag
}

// GenerateEntityTag generates the value to return in the "ETag" HTTP response header for the given entity
// The ETag is the base64-encoded value of the md5 hash of the buffer content
func GenerateEntityTag(entity ConditionalRequestEntity) string {
	var buffer bytes.Buffer
	buffer.WriteString(generateETagValue(entity.GetETagData()))
	etagData := md5.Sum(buffer.Bytes())
	etag := base64.StdEncoding.EncodeToString(etagData[:])
	return etag
}

// GenerateEntitiesTag generates the value to return in the "ETag" HTTP response header for the given list of entities
// The ETag is the base64-encoded value of the md5 hash of the buffer content
func GenerateEntitiesTag(entities []ConditionalRequestEntity) string {
	var buffer bytes.Buffer
	for i, entity := range entities {
		buffer.WriteString(generateETagValue(entity.GetETagData()))
		if i < len(entities)-1 {
			buffer.WriteString("\n")
		}
	}
	etagData := md5.Sum(buffer.Bytes())
	etag := base64.StdEncoding.EncodeToString(etagData[:])
	return etag
}
func generateETagValue(data []interface{}, options ...interface{}) string {
	var buffer bytes.Buffer
	for i, d := range data {
		switch d := d.(type) {
		case []interface{}:
			// if the entry in the 'data' array is itself an array,
			// then we recursively call the 'generateETagValue' function with this array entry.
			buffer.WriteString(generateETagValue(d))
		case string:
			buffer.WriteString(d)
		case *string:
			if d != nil {
				buffer.WriteString(*d)
			}
		case time.Time:
			buffer.WriteString(d.UTC().String())
		case *time.Time:
			if d != nil {
				buffer.WriteString(d.UTC().String())
			}
		case int:
			buffer.WriteString(strconv.Itoa(d))
		case *int:
			if d != nil {
				buffer.WriteString(strconv.Itoa(*d))
			}
		case uuid.UUID:
			buffer.WriteString(d.String())
		case *uuid.UUID:
			if d != nil {
				buffer.WriteString(d.String())
			}
		default:
			log.Logger().Errorln(fmt.Sprintf("Unexpected Etag fragment format: %v", reflect.TypeOf(d)))
		}
		if i < len(data)-1 {
			buffer.WriteString("|")
		}
	}
	return buffer.String()
}

// modifiedSince compares the given context's 'IfModifiedSince' value is before the given 'lastModified' argument
// Returns 'true, true' if the 'If-Modified' field was found and matched the given 'lastModified' argument
// Returns 'true, false' if the 'If-Modified' field was found but did not match with given 'lastModified' argument
// Returns 'false, false' if the 'If-Modified' field was not found
func modifiedSince(ctx ConditionalRequestContext, lastModified time.Time) (bool, bool) {
	if ctx.getIfModifiedSince() != nil {
		ifModifiedSince := *ctx.getIfModifiedSince()
		// 'If-Modified' field was found and matched the given 'lastModified' argument
		if ifModifiedSince.UTC().Truncate(time.Second).Before(lastModified.UTC().Truncate(time.Second)) {
			return true, true
		}
		// 'If-Modified' field was found but did not match with given 'lastModified' argument
		return true, false
	}
	// 'If-Modified' field was not found
	return false, false
}

// matchesETag compares the given 'etag' argument matches with the context's 'IfNoneMatch' value.
// Returns 'true, true' if the 'If-None-Match' field was found and matched given 'etag' argument
// Returns 'true, false' if the 'If-None-Match' field was found but did not match given 'etag' argument
// Returns 'false, false' if the 'If-None-Match' field was not found
func matchesETag(ctx ConditionalRequestContext, etag string) (bool, bool) {
	if ctx.getIfNoneMatch() != nil {
		if *ctx.getIfNoneMatch() == etag {
			// 'If-None-Match' field was found and matched the given 'etag' argument
			return true, true
		}
		// 'If-None-Match' field was found and but did not match the given 'etag' argument
		return true, false
	}
	// 'If-None-Match' field was not found
	return false, false
}

// ToHTTPTime utility function to convert a 'time.Time' into a valid HTTP date
func ToHTTPTime(value time.Time) string {
	return value.UTC().Format(http.TimeFormat)
}

// ConditionalEntities checks if the entities to return changed since the client's last call and returns a "304 Not Modified" response
// or calls the 'nonConditionalCallback' function to carry on.
func (ctx *WorkItemsResourceListContext) ConditionalEntities(entities []workitemdsl.WorkItem, cacheControlConfig CacheControlConfig, nonConditionalCallback func()) {
	conditionalEntities := make([]ConditionalRequestEntity, len(entities))
	for i, entity := range entities {
		conditionalEntities[i] = entity
	}
	doConditionalEntities(ctx, conditionalEntities, cacheControlConfig, nonConditionalCallback)
}

// getIfModifiedSince sets the 'If-Modified-Since' header
func (ctx *WorkItemsResourceListContext) getIfModifiedSince() *time.Time {
	ifModifiedSince := ctx.Request.Header.Get(IfModifiedSince)
	if ifModifiedSince != "" {
		val, _ := http.ParseTime(ifModifiedSince)
		return &val
	}
	return nil
}

// SetLastModified sets the 'Last-Modified' header
func (ctx *WorkItemsResourceListContext) setLastModified(value time.Time) {
	ctx.Header(LastModified, ToHTTPTime(value))
}

// getIfNoneMatch sets the 'If-None-Match' header
func (ctx *WorkItemsResourceListContext) getIfNoneMatch() *string {
	ifNoneMatch := ctx.Request.Header.Get(IfNoneMatch)
	if ifNoneMatch != "" {
		return &ifNoneMatch
	}
	return nil
}

// setETag sets the 'ETag' header
func (ctx *WorkItemsResourceListContext) setETag(value string) {
	ctx.Header(ETag, value)
}

// SetCacheControl sets the 'Cache-Control' header
func (ctx *WorkItemsResourceListContext) setCacheControl(value string) {
	ctx.Header(CacheControl, value)
}
