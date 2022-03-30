#!/bin/bash

rm -rf ./output
mkdir output

for number in {0..20}
do
./nftgen generate --output $number.png
mv $number.png output/$number.png
mv $number.json output/$number.json
echo "Generated: $number"
done

exit 0

