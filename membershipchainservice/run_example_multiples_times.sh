#!/bin/bash
for i in {39..100}
do
  echo "$i" > folder_str
  echo "Starting $i"
  echo "_____________________________________________________________________________"
  mkdir log
  mkdir "Data/$i/"
  mkdir "Data/$i/Random"
  mkdir "Data/$i/Locarno"
  ulimit -n 100000000
  ulimit -n
  go test -v -run "WholeSystem" 2>&1 | tee "log/$i"
done
rm folder_name
