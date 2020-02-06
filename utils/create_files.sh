#!/bin/bash
for i in {64...100}
do
   echo "$i" > folder_name
   mkdir "NodesFiles/$i"
   mkdir "PingsFiles/$i"
   ./generate_nodes.py
   ls "NodesFiles/$i"
 done
rm folder_name
