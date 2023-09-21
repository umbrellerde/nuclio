#!/usr/bin/env bash

functions=( check virus ocr email urgentcheck urgentvirus urgentocr urgentemail )

mkdir -p "logs"

for name in "${functions[@]}"
do
  docker logs "nuclio-nuclio-$name"> "logs/$name.log" 2>&1
done

docker logs "nuclio-dashboard"> "logs/nuclio-dashboard.log" 2>&1