#!/bin/bash

set -e -u

cd $(dirname $0)

for resource in $(cat resources); do
  echo "checking $resource..."

  mkdir -p /tmp/resources/$resource
  cd /tmp/resources/$resource
  if [ -d .git ]; then
    git pull
  else
    git clone https://github.com/concourse/$resource-resource .
  fi

  if git log $(git describe --tags --abbrev=0)...master \
    --oneline --color | \
    grep --color=never .; then
    echo $'\e[31mhas changes!\e[0m'
  else
    echo $'\e[32mok\e[0m'
  fi

  echo
done
