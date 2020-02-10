#!/bin/bash
for i in {2..10}
do
  ssh apannati@users.deterlab.net ssh site-1.NyleLocarnoLottery.SAFER echo "$i" > remote/folder_str
  echo "Starting $i"
  echo "_____________________________________________________________________________"
  ./simulation -platform=deterlab locarno.toml
done
./copy_data_locarno.sh
rm folder_name
