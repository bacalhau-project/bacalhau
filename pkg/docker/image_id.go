package docker

import (
	"fmt"
	"strings"

	"github.com/docker/distribution/reference"
)

const (
	// Durations in seconds
	oneHour  = 3600
	oneDay   = oneHour * 24
	oneMonth = oneDay * 28
)

type EmptyTag struct{}
type NameTag string
type DigestTag string

type ImageTag interface {
	CacheDuration() int64
	Separator() string
	String() string
}

func (EmptyTag) Separator() string    { return "" }
func (EmptyTag) String() string       { return "" }
func (EmptyTag) CacheDuration() int64 { return oneHour }

func (NameTag) Separator() string { return ":" }
func (n NameTag) String() string  { return string(n) }
func (n NameTag) CacheDuration() int64 {
	if n.String() == "latest" {
		return oneHour
	}
	return oneDay
}

func (DigestTag) Separator() string    { return "@" }
func (n DigestTag) String() string     { return string(n) }
func (DigestTag) CacheDuration() int64 { return oneMonth }

func ParseImageTag(s string) (string, ImageTag) {
	// Check for a digest, and if not look for a tag
	if strings.Contains(s, "@") {
		tagSplit := strings.Split(s, "@")
		return tagSplit[0], DigestTag(tagSplit[1])
	} else if strings.Contains(s, ":") {
		tagSplit := strings.Split(s, ":")
		return tagSplit[0], NameTag(tagSplit[1])
	}

	return s, EmptyTag{}
}

type ImageID struct {
	// e.g. ghcr.io
	repository string

	// e.g. ubuntu or organization/ubuntu or organization/user/ubuntu
	name string

	// e.g. latest or kinetic or @sha256:digest-string
	tag ImageTag
}

func NewImageID(imageID string) (*ImageID, error) {
	id := &ImageID{}

	repo, err := reference.Parse(imageID)
	if err != nil {
		return nil, err
	}

	if named, ok := repo.(reference.Named); ok {
		id.repository = reference.Domain(named)
		id.name = reference.Path(named)
	}

	if digested, ok := repo.(reference.Digested); ok {
		obj := digested.Digest()
		digest := fmt.Sprintf("%s:%s", obj.Algorithm().String(), obj.Encoded())
		id.tag = DigestTag(digest)
	} else if tagged, ok := repo.(reference.Tagged); ok {
		id.tag = NameTag(tagged.Tag())
	} else {
		id.tag = EmptyTag{}
	}

	return id, nil
}

func (i *ImageID) HasDigest() bool {
	_, ok := i.tag.(DigestTag)
	return ok
}

func (i *ImageID) String() string {
	name := fmt.Sprintf("%s%s%s", i.name, i.tag.Separator(), i.tag.String())
	if i.repository != "" {
		name = fmt.Sprintf("%s/%s", i.repository, name)
	}
	return name
}
