package testhelper

import "fmt"

func CombineConfigs(configs ...string) string {
	var config string
	for _, c := range configs {
		config = fmt.Sprintf("%s\n%s", config, c)
	}
	return config
}

func ConfigDataDC1() string {
	return fmt.Sprintf(`
data "vsphere_datacenter" "dc1" {
  name = vsphere_datacenter.dc1.name
}
`)
}

func ConfigDataDC2() string {
	return fmt.Sprintf(`
data "vsphere_datacenter" "dc2" {
  name = vsphere_datacenter.dc2.name
}
`)
}

func ConfigResDC1() string {
	return fmt.Sprintf(`
resource "vsphere_datacenter" "dc1" {
  name = "testdc1"
  tags = [vsphere_tag.tag1.id, vsphere_tag.tag2.id]
}
`)
}

func ConfigResDC2() string {
	return fmt.Sprintf(`
resource "vsphere_datacenter" "dc2" {
  name = "testdc2"
  tags = [ vsphere_tag.tag1.id, vsphere_tag.tag3.id ]
}
`)
}

func ConfigResTagCat1() string {
	return fmt.Sprintf(`
resource "vsphere_tag_category" "category1" {
  name        = "cat1"
  description = "cat1"
  cardinality = "MULTIPLE"

  associable_types = [
    "Datacenter"
  ]
}
`)
}

func ConfigResTagCat2() string {
	return fmt.Sprintf(`
resource "vsphere_tag_category" "category2" {
  name        = "cat2"
  description = "cat2"
  cardinality = "MULTIPLE"

  associable_types = [
    "Datacenter"
  ]
}
`)
}

func ConfigResTag1() string {
	return fmt.Sprintf(`
resource "vsphere_tag" "tag1" {
  name        = "tag1"
  category_id = vsphere_tag_category.category1.id
}
`)
}

func ConfigResTag2() string {
	return fmt.Sprintf(`
resource "vsphere_tag" "tag2" {
  name        = "tag2"
  category_id = vsphere_tag_category.category2.id
}
`)
}

func ConfigResTag3() string {
	return fmt.Sprintf(`
resource "vsphere_tag" "tag3" {
  name        = "tag2"
  category_id = vsphere_tag_category.category1.id
}`)
}
