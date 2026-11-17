# Device authorization can be imported using the node ID (preferred) or the
# legacy ID. An imported device keeps delete_on_destroy = false until the
# configuration sets it and is applied.
import {
  to = tailscale_device_authorization.old_node
  id = "nodeidCNTRL"
}
