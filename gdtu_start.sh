#!/bin/bash
nohup /Users/liuwei/Desktop/fengzhiyou/eth-clone/go-gdtu/build/bin/ggdtu --port 30001 --unlock "0x23D67e6041639123D7C635eC36AD57487b64c8FE" --password "/Users/liuwei/Desktop/fengzhiyou/eth-clone/go-gdtu/build/bin/pass.txt" --mine --miner.gdtuerbase 0 --nodiscover > /Users/liuwei/Desktop/fengzhiyou/eth-clone/go-gdtu/build/bin/log/ggdtu_`date +'%Y-%m-%d'`.log 2>&1 &
