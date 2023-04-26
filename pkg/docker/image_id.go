package docker

import (
	"fmt"
	"strings"

	"github.com/docker/distribution/reference"
)

type EmptyTag struct{}
type NameTag string
type DigestTag string

type ImageTag interface {
	Separator() string
	String() string
}

func (EmptyTag) Separator() string { return "" }
func (EmptyTag) String() string    { return "" }

func (NameTag) Separator() string { return ":" }
func (n NameTag) String() string  { return string(n) }

func (DigestTag) Separator() string { return "@" }
func (n DigestTag) String() string  { return string(n) }

func ParseImageTag(s string) (string, ImageTag) {
	// Check for a digest, and if not look for a tag
	if strings.Contains(s, "@") {
		tag_split := strings.Split(s, "@")
		return tag_split[0], DigestTag(tag_split[1])
	} else if strings.Contains(s, ":") {
		tag_split := strings.Split(s, ":")
		return tag_split[0], NameTag(tag_split[1])
	}

	return s, EmptyTag{}
}

type ImageID struct {
	// e.g. ghcr.io
	repository string

	// e.g. ubuntu or organisation/ubuntu or organisation/user/ubuntu
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
		digest := fmt.Sprintf("sha256:%s", digested.Digest().Encoded())
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
