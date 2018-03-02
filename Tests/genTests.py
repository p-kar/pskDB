import sys
import random

n_server = int(sys.argv[1]) 
n_client = int(sys.argv[2])

# Setup servers
joinServerRequests = []
for i in xrange(n_server):
  joinServerRequests.append("joinServer " + str(i))

#Setup clients
joinClientRequests = []
for j in xrange(n_client):
  joinClientRequests.append("joinClient " + str(j + i + 1) + " " + str(random.randint(0, n_server - 1))) 

n_keys = 40
key_req = ["get", "put"]

Requests = []
n_reqs = int(sys.argv[3])
for m in xrange(n_reqs):
  if m % 900 == 0:
    y = random.randint(0, n_server - 1)
    Requests.append("breakConnection " + str(y) + " " + str(n_server - y - 1))
  elif m % 400 == 0:
    Requests.append("stabilize")
  else :
    req_type = key_req[random.randint(0,1)]
    if req_type == "get" :
      Requests.append("get " + str(n_server + random.randint(0, n_client - 1)) + " " + str(random.randint(1,40)))
    else : 
      Requests.append("put " + str(n_server + random.randint(0, n_client - 1)) + " " + str(random.randint(1,40)) + " " + random.choice('abcdefghijhijklmnopqrstuvwxyz'))

for s in joinServerRequests:
  print s

for c in joinClientRequests:
  print c

random.shuffle(Requests)

for x in Requests:
  print x

print "done"