#!/usr/bin/env bash
# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

docs=$(find ./docs -type f -name "*.md")
error=false

for doc in $docs; do
  dirname=$(dirname "$doc")
  category=$(basename "$dirname")


  case "$category" in
    "guides")
      # Guides have no requirements.
      continue
      ;;

    "data-sources")
      # Data sources require a subcategory.
      grep "^subcategory: " "$doc" > /dev/null
      if [[ "$?" == "1" ]]; then
        echo "Data source documentation is missing a subcategory: $doc"
        error=true
      fi
      ;;

		"resources")
      # Resources require a subcategory.
      grep "^subcategory: " "$doc" > /dev/null
      if [[ "$?" == "1" ]]; then
        echo "Resource documentation is missing a subcategory: $doc"
        error=true
      fi
      ;;

    *)
      continue
      ;;
  esac
done

if $error; then
  exit 1
fi

echo "==> Done."
exit 0
