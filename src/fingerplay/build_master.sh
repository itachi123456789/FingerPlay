#! /bin/bash

dest_path=/home/pandazhong/workspace/bigplayer/fingerplay-bin

echo "go building..."

go build

if [ $? != 0 ]; then 
    echo "go build failed"
    exit 1
fi

echo "go build ok"

pushd $dest_path
git checkout master
popd

cp fingerplay $dest_path/bin/
cp conf/fingerplay-develop.toml $dest_path/conf/
cp conf/fingerplay-staging.toml $dest_path/conf/
cp conf/fingerplay-master.toml $dest_path/conf/

pushd $dest_path
git add -A .
git commit -m "Add Or Remove Or Update..."
git push origin master
popd

exit $?
