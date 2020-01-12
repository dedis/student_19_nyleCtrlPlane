#!/bin/bash
cd DATASET
curl "https://opentransportdata.swiss/dataset/9ee205ba-7bda-46bb-918d-30f0522372cf/resource/ef864b35-8995-4f20-924f-01938ed3bd5c/download/4tuchoevsammlungch201920180528031501.zip" --output data.zip
unzip data.zip
mv DURCHBI A_DURCHBI
mv METABHF A_METABHF
rm data.zip
cd ..
