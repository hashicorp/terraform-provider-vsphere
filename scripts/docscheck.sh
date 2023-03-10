#!/usr/bin/env bash
# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0


docs=$(ls website/docs/**/*.markdown)
error=false

for doc in $docs; do
  dirname=$(dirname "$doc")
  category=$(basename "$dirname")


  case "$category" in
    "guides")
      # Guides have no requirements.
      continue
      ;;

    "d")
      # Data sources require a subcategory.
      grep "^subcategory: " "$doc" > /dev/null
      if [[ "$?" == "1" ]]; then
        echo "Data source documentation is missing a subcategory: $doc"
        error=true
      fi
      ;;

		"r")
      # Resources require a subcategory.
      grep "^subcategory: " "$doc" > /dev/null
      if [[ "$?" == "1" ]]; then
        echo "Resource documentation is missing a subcategory: $doc"
        error=true
      fi
      ;;

    *)
      # Docs
      error=true
      echo "Unknown category \"$category\". " \
        "Documentation can only exist in r/, d/, or guides/ directories."
      ;;
  esac
done

if $error; then
  exit 1
fi

exit 0
