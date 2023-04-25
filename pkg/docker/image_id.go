package docker

import (
	"fmt"
	"strings"
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

func (n NameTag) String() string { return string(n) }

func (DigestTag) Separator() string { return "@" }

func (n DigestTag) String() string { return string(n) }

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

func NewImageID(imageID string) ImageID {
	id := ImageID{}

	parts := strings.Split(imageID, "/")

	// Check the head for what looks like a domain, e.g. ghcr.io
	if strings.Contains(parts[0], ".") {
		id.repository = parts[0]
	}

	// Parse the name and tag from the last element. Once done
	// we will replace the last element with the name (so that
	// we can use it later).
	id.name, id.tag = ParseImageTag(parts[len(parts)-1])
	parts[len(parts)-1] = id.name

	// Strip the repository off the front if it exists so we
	// can safely join the remaining parts to get the name
	if id.repository != "" {
		parts = parts[1:]
	}

	id.name = strings.Join(parts, "/")

	return id
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
