#!/bin/bash
echo "Remove DataBase"
ssh apannati@users.deterlab.net ssh server-0.NyleMemberGraphs.SAFER sudo find remote -maxdepth 1 -name '*.db' -delete 
echo "Change Rights"
ssh apannati@users.deterlab.net ssh server-0.NyleMemberGraphs.SAFER sudo chmod 777 remote/Data/\*/\*.txt
ssh apannati@users.deterlab.net ssh server-0.NyleMemberGraphs.SAFER sudo chmod 777 remote/Data/\*.txt
echo "Deleting useless files"
ssh apannati@users.deterlab.net ssh server-0.NyleMemberGraphs.SAFER sudo rm -rf remote/Data/Random/
echo "TAR"
ssh apannati@users.deterlab.net ssh server-0.NyleMemberGraphs.SAFER tar -zcvf remote/data.tar.gz remote/Data
scp apannati@users.deterlab.net:remote/data.tar.gz data.tar.gz
rm -rf Data
tar -zxvf data.tar.gz
echo "Merge files"
mv remote/Data Data
rmdir remote
find Data/Throughput -maxdepth 1 -type f -name '*.txt' -print0 | sort -z | xargs -0 cat -- >> Data/Throughput.txt
rm -rf Data/Throughput/
echo "Deleting archive and Clearing"
ssh apannati@users.deterlab.net ssh server-0.NyleMemberGraphs.SAFER sudo rm remote/data.tar.gz
rm data.tar.gz
ssh apannati@users.deterlab.net ssh server-0.NyleMemberGraphs.SAFER sudo rm -rf remote/Data/Throughput
ssh apannati@users.deterlab.net ssh server-0.NyleMemberGraphs.SAFER sudo mkdir remote/Data/Throughput
echo "Archive"
NUM=`ls Archive/arch* | sed -n 's/Archive\/arch\([0-9]*\).txt/\1/p' | sort -rh | head -n 1`
cp Data/Throughput.txt Archive/arch$((NUM + 1)).txt