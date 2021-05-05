// This test checks basic subscription support.

--> {"jsonrpc":"2.0","id":1,"Method":"nftest_subscribe","params":["someSubscription",5,1]}
<-- {"jsonrpc":"2.0","id":1,"result":"gd1"}
<-- {"jsonrpc":"2.0","Method":"nftest_subscription","params":{"subscription":"gd1","result":1}}
<-- {"jsonrpc":"2.0","Method":"nftest_subscription","params":{"subscription":"gd1","result":2}}
<-- {"jsonrpc":"2.0","Method":"nftest_subscription","params":{"subscription":"gd1","result":3}}
<-- {"jsonrpc":"2.0","Method":"nftest_subscription","params":{"subscription":"gd1","result":4}}
<-- {"jsonrpc":"2.0","Method":"nftest_subscription","params":{"subscription":"gd1","result":5}}

--> {"jsonrpc":"2.0","id":2,"Method":"nftest_echo","params":[11]}
<-- {"jsonrpc":"2.0","id":2,"result":11}
