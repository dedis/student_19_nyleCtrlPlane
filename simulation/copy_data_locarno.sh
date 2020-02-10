#!/bin/bash
echo "Remove DataBase"
ssh apannati@users.deterlab.net ssh site-1.NyleLocarnoLottery.SAFER sudo find remote -maxdepth 1 -name '*.db' -delete 
echo "Change Rights"
ssh apannati@users.deterlab.net ssh site-1.NyleLocarnoLottery.SAFER sudo chmod 777 remote/Data/\*/\*.txt
ssh apannati@users.deterlab.net ssh site-1.NyleLocarnoLottery.SAFER sudo chmod 777 remote/Data/\*.txt
echo "Deleting useless files"
ssh apannati@users.deterlab.net ssh site-1.NyleLocarnoLottery.SAFER sudo rm -rf remote/Data/Random/
echo "TAR"
ssh apannati@users.deterlab.net ssh site-1.NyleLocarnoLottery.SAFER tar -zcvf remote/data.tar.gz remote/Data
scp apannati@users.deterlab.net:remote/data.tar.gz data.tar.gz
rm -rf Data
tar -zxvf data.tar.gz
echo "Merge files"
mv remote/Data Data
rmdir remote
ssh apannati@users.deterlab.net ssh site-1.NyleLocarnoLottery.SAFER sudo rm remote/data.tar.gz
rm data.tar.gz
ssh apannati@users.deterlab.net ssh site-1.NyleLocarnoLottery.SAFER sudo rm -rf remote/Data/
