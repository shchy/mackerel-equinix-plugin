docker run \
  -v /var/run/docker.sock:/var/run/docker.sock \
  -v $PWD/mackerel-agent.conf:/etc/mackerel-agent/mackerel-agent.conf \
  -v $PWD/conf.d:/etc/mackerel-agent/conf.d:ro \
  -v $PWD/mackerel-equinix-plugin:/etc/mackerel-agent/mackerel-equinix-plugin \
  --name mackerel-agent2 \
  -d \
  mackerel/mackerel-agent


#   -v $PWD/var/lib/mackerel-agent/:/var/lib/mackerel-agent/ \