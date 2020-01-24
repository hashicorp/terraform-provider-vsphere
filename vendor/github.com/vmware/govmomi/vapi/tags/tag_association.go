/*
Copyright (c) 2018 VMware, Inc. All Rights Reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

vUnless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package tags

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/vmware/govmomi/vapi/internal"
	"github.com/vmware/govmomi/vim25/mo"
)

func (c *Manager) tagID(ctx context.Context, id string) (string, error) {
	if isName(id) {
		tag, err := c.GetTag(ctx, id)
		if err != nil {
			return "", err
		}
		return tag.ID, nil
	}
	return id, nil
}

// AttachTag attaches a tag ID to a managed object.
func (c *Manager) AttachTag(ctx context.Context, tagID string, ref mo.Reference) error {
	id, err := c.tagID(ctx, tagID)
	if err != nil {
		return err
	}
	spec := internal.NewAssociation(ref)
	url := internal.URL(c, internal.AssociationPath).WithID(id).WithAction("attach")
	return c.Do(ctx, url.Request(http.MethodPost, spec), nil)
}

// DetachTag detaches a tag ID from a managed object.
// If the tag is already removed from the object, then this operation is a no-op and an error will not be thrown.
func (c *Manager) DetachTag(ctx context.Context, tagID string, ref mo.Reference) error {
	id, err := c.tagID(ctx, tagID)
	if err != nil {
		return err
	}
	spec := internal.NewAssociation(ref)
	url := internal.URL(c, internal.AssociationPath).WithID(id).WithAction("detach")
	return c.Do(ctx, url.Request(http.MethodPost, spec), nil)
}

// ListAttachedTags fetches the array of tag IDs attached to the given object.
func (c *Manager) ListAttachedTags(ctx context.Context, ref mo.Reference) ([]string, error) {
	spec := internal.NewAssociation(ref)
	url := internal.URL(c, internal.AssociationPath).WithAction("list-attached-tags")
	var res []string
	return res, c.Do(ctx, url.Request(http.MethodPost, spec), &res)
}

// GetAttachedTags fetches the array of tags attached to the given object.
func (c *Manager) GetAttachedTags(ctx context.Context, ref mo.Reference) ([]Tag, error) {
	ids, err := c.ListAttachedTags(ctx, ref)
	if err != nil {
		return nil, fmt.Errorf("get attached tags %s: %s", ref, err)
	}

	var info []Tag
	for _, id := range ids {
		tag, err := c.GetTag(ctx, id)
		if err != nil {
			return nil, fmt.Errorf("get tag %s: %s", id, err)
		}
		info = append(info, *tag)
	}
	return info, nil
}

// ListAttachedObjects fetches the array of attached objects for the given tag ID.
func (c *Manager) ListAttachedObjects(ctx context.Context, tagID string) ([]mo.Reference, error) {
	id, err := c.tagID(ctx, tagID)
	if err != nil {
		return nil, err
	}
	url := internal.URL(c, internal.AssociationPath).WithID(id).WithAction("list-attached-objects")
	var res []internal.AssociatedObject
	if err := c.Do(ctx, url.Request(http.MethodPost, nil), &res); err != nil {
		return nil, err
	}

	refs := make([]mo.Reference, len(res))
	for i := range res {
		refs[i] = res[i]
	}
	return refs, nil
}

// AttachedObjects is the response type used by ListAttachedObjectsOnTags.
type AttachedObjects struct {
	TagID     string         `json:"tag_id"`
	Tag       *Tag           `json:"tag,omitempty"`
	ObjectIDs []mo.Reference `json:"object_ids"`
}

// UnmarshalJSON implements json.Unmarshaler.
func (t *AttachedObjects) UnmarshalJSON(b []byte) error {
	var o struct {
		TagID     string                      `json:"tag_id"`
		ObjectIDs []internal.AssociatedObject `json:"object_ids"`
	}
	err := json.Unmarshal(b, &o)
	if err != nil {
		return err
	}

	t.TagID = o.TagID
	t.ObjectIDs = make([]mo.Reference, len(o.ObjectIDs))
	for i := range o.ObjectIDs {
		t.ObjectIDs[i] = o.ObjectIDs[i]
	}

	return nil
}

// ListAttachedObjectsOnTags fetches the array of attached objects for the given tag IDs.
func (c *Manager) ListAttachedObjectsOnTags(ctx context.Context, tagID []string) ([]AttachedObjects, error) {
	var ids []string
	for i := range tagID {
		id, err := c.tagID(ctx, tagID[i])
		if err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}

	spec := struct {
		TagIDs []string `json:"tag_ids"`
	}{ids}

	url := internal.URL(c, internal.AssociationPath).WithAction("list-attached-objects-on-tags")
	var res []AttachedObjects
	return res, c.Do(ctx, url.Request(http.MethodPost, spec), &res)
}

// GetAttachedObjectsOnTags combines ListAttachedObjectsOnTags and populates each Tag field.
func (c *Manager) GetAttachedObjectsOnTags(ctx context.Context, tagID []string) ([]AttachedObjects, error) {
	objs, err := c.ListAttachedObjectsOnTags(ctx, tagID)
	if err != nil {
		return nil, fmt.Errorf("list attached objects %s: %s", tagID, err)
	}

	tags := make(map[string]*Tag)

	for i := range objs {
		var err error
		id := objs[i].TagID
		tag, ok := tags[id]
		if !ok {
			tag, err = c.GetTag(ctx, id)
			if err != nil {
				return nil, fmt.Errorf("get tag %s: %s", id, err)
			}
			objs[i].Tag = tag
		}
	}

	return objs, nil
}

// AttachedTags is the response type used by ListAttachedTagsOnObjects.
type AttachedTags struct {
	ObjectID mo.Reference `json:"object_id"`
	TagIDs   []string     `json:"tag_ids"`
	Tags     []Tag        `json:"tags,omitempty"`
}

// UnmarshalJSON implements json.Unmarshaler.
func (t *AttachedTags) UnmarshalJSON(b []byte) error {
	var o struct {
		ObjectID internal.AssociatedObject `json:"object_id"`
		TagIDs   []string                  `json:"tag_ids"`
	}
	err := json.Unmarshal(b, &o)
	if err != nil {
		return err
	}

	t.ObjectID = o.ObjectID
	t.TagIDs = o.TagIDs

	return nil
}

// ListAttachedTagsOnObjects fetches the array of attached tag IDs for the given object IDs.
func (c *Manager) ListAttachedTagsOnObjects(ctx context.Context, objectID []mo.Reference) ([]AttachedTags, error) {
	var ids []internal.AssociatedObject
	for i := range objectID {
		ids = append(ids, internal.AssociatedObject(objectID[i].Reference()))
	}

	spec := struct {
		ObjectIDs []internal.AssociatedObject `json:"object_ids"`
	}{ids}

	url := internal.URL(c, internal.AssociationPath).WithAction("list-attached-tags-on-objects")
	var res []AttachedTags
	return res, c.Do(ctx, url.Request(http.MethodPost, spec), &res)
}

// GetAttachedTagsOnObjects calls ListAttachedTagsOnObjects and populates each Tags field.
func (c *Manager) GetAttachedTagsOnObjects(ctx context.Context, objectID []mo.Reference) ([]AttachedTags, error) {
	objs, err := c.ListAttachedTagsOnObjects(ctx, objectID)
	if err != nil {
		return nil, fmt.Errorf("list attached tags %s: %s", objectID, err)
	}

	tags := make(map[string]*Tag)

	for i := range objs {
		for _, id := range objs[i].TagIDs {
			var err error
			tag, ok := tags[id]
			if !ok {
				tag, err = c.GetTag(ctx, id)
				if err != nil {
					return nil, fmt.Errorf("get tag %s: %s", id, err)
				}
				tags[id] = tag
			}
			objs[i].Tags = append(objs[i].Tags, *tag)
		}
	}

	return objs, nil
}
