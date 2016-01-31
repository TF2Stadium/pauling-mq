pauling-mq
==========

A "message queue" for Pauling, queues messages when pauling is offline, then sends 
them back to Pauling once it's online.

# Configuration
pauling-mq is controlled via these environment variables:
* `PAULING_RPC_ADDR` - RPC address for Pauling
* `PAULING_LOGS_ADDR` - address where Pauling is listening for UDP addresses.
* `LOGS_ADDR` - address on which logs are received.
