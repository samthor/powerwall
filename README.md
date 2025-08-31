Go library for talking to Powerwall devices (tested on 3, likely works on 2 as well).

Requires network access to the Powerwall's private IP, aka 192.168.91.1.
Note that as of ~2025, you must appear to be on the 192.168.91.x subnet for this to work (otherwise you'll get permission denied errors, even with the correct password).

## Usage

You can create a `powerwall.TEDApi` and call queries directly, or use the `powerwall.SimpleStatus` helper.

```go
func main() {
  api := powerwall.TEDApi{Secret: "..."}

  // do raw query
  raw, err := api.Query(context.TODO(), powerwall.QueryStatus)
  // ...do something with raw JSON

  // or this helper
  status, err := powerwall.GetSimpleStatus(context.TODO(), api)
  // ...
}
```

## Network Access

You need access to the Powerwall's private IP/network for calls to succeed.

One way to do this is put a device like a rasberry PI on your Ethernet network, and have it join the Powerwall's WiFi, and do your calls there.

### Sam's Network

I use my router to connect to the Powerwall, rather than a smart intermediary device.
I use Unifi hardware, which makes this pretty easy.

- I have a VLAN (with ID 91, just for convenience) just for the Powerwall
- A port on that VLAN has a [WiFi dongle](https://www.tp-link.com/au/home-networking/wifi-router/tl-wr802n/) that is a dumb client joining to the Powerwall's internal WiFi ("TeslaPW_xxxxx")
- There's a rule under "Network > Settings > Policy Engine > Policy Table" that looks like [this](policy-table-rule.png): basically, make all traffic to 192.168.91.x look like it comes from the router

With that, the Powerwall thinks we're connecting from the right subnet.

The TP-Link dongle isn't that good and occasionally crashes.
In my house, it's powered by PoE (PoE => microUSB adaptor) and it can be restarted via the Unifi console or its API.

## Thanks

This project uses the proto files from [pypowerwall](https://github.com/jasonacox/pypowerwall).
Their LICENSE file is inside third_party/.

In case you asked, I just don't like Python.
It's a good learning language but an absolute pain to use in any real-world situation.
