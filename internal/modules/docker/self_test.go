package docker

import "testing"

func TestContainerIDPatterns(t *testing.T) {
	const real = "4bff9c70c35e07d652a11fa9b836dcbe6ecab280274226130ef23436d851f34a"
	const layer = "7c8e6bf192e84d2b61d8374d9959204c04696bf0312d03ab3c483d327fd9ccc6"
	mountinfo := "4119 2049 0:144 / / rw - overlay overlay rw,upperdir=/vol1/docker/overlay2/" + layer + "/diff,workdir=/vol1/docker/overlay2/" + layer + "/work\n" +
		"4128 4119 0:56 /docker/containers/" + real + "/resolv.conf /etc/resolv.conf rw - btrfs /dev/x rw\n"
	var got string
	for _, re := range containerIDPatterns {
		if m := re.FindStringSubmatch(mountinfo); m != nil {
			got = m[1]
			break
		}
	}
	if got != real {
		t.Fatalf("mountinfo 解析得到 %q，期望 %q", got, real)
	}
	cg := "0::/system.slice/docker-" + real + ".scope\n"
	got = ""
	for _, re := range containerIDPatterns {
		if m := re.FindStringSubmatch(cg); m != nil {
			got = m[1]
			break
		}
	}
	if got != real {
		t.Fatalf("cgroup 解析得到 %q，期望 %q", got, real)
	}
}
