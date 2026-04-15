<p>Packages:</p>
<ul>
<li>
<a href="#mirror.extensions.gardener.cloud%2fv1alpha1">mirror.extensions.gardener.cloud/v1alpha1</a>
</li>
</ul>

<h2 id="mirror.extensions.gardener.cloud/v1alpha1">mirror.extensions.gardener.cloud/v1alpha1</h2>
<p>

</p>

<h3 id="mirrorconfig">MirrorConfig
</h3>


<p>
MirrorConfig contains information about registry mirrors to configure.
</p>

<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>

<tr>
<td>
<code>mirrors</code></br>
<em>
<a href="#mirrorconfiguration">MirrorConfiguration</a> array
</em>
</td>
<td>
<p>Mirrors is a slice of registry mirrors to configure.</p>
</td>
</tr>

</tbody>
</table>


<h3 id="mirrorconfiguration">MirrorConfiguration
</h3>


<p>
(<em>Appears on:</em><a href="#mirrorconfig">MirrorConfig</a>)
</p>

<p>
MirrorConfiguration represents a registry mirror.
</p>

<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>

<tr>
<td>
<code>upstream</code></br>
<em>
string
</em>
</td>
<td>
<p>Upstream is the remote registry host to mirror.<br />The value must be a valid DNS subdomain (RFC 1123) and optionally a port.</p>
</td>
</tr>
<tr>
<td>
<code>hosts</code></br>
<em>
<a href="#mirrorhost">MirrorHost</a> array
</em>
</td>
<td>
<p>Hosts are the mirror hosts to be used for the upstream.</p>
</td>
</tr>

</tbody>
</table>


<h3 id="mirrorhost">MirrorHost
</h3>


<p>
(<em>Appears on:</em><a href="#mirrorconfiguration">MirrorConfiguration</a>)
</p>

<p>
MirrorHost represents a mirror host.
</p>

<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>

<tr>
<td>
<code>host</code></br>
<em>
string
</em>
</td>
<td>
<p>Host is the mirror host.</p>
</td>
</tr>
<tr>
<td>
<code>capabilities</code></br>
<em>
<a href="#mirrorhostcapability">MirrorHostCapability</a> array
</em>
</td>
<td>
<em>(Optional)</em>
<p>Capabilities are the operations a host is capable of performing.<br />This also represents the set of operations for which the mirror host may be trusted to perform.<br />The supported values are "pull" and "resolve".<br />Defaults to ["pull"].</p>
</td>
</tr>
<tr>
<td>
<code>caBundleSecretReferenceName</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>CABundleSecretReferenceName is the reference name for a Secret containing a PEM-encoded certificate authority bundle.<br />The CA bundle is used to verify the TLS certificate of the mirror host.<br />The referenced secret must be immutable and must have a data key `bundle.crt`.</p>
</td>
</tr>
<tr>
<td>
<code>overridePath</code></br>
<em>
boolean
</em>
</td>
<td>
<em>(Optional)</em>
<p>OverridePath represents the `override_path` field in the hosts.toml file for containerd registry configuration.<br />See https://github.com/containerd/containerd/blob/v2.2.0/docs/hosts.md#override_path-field<br />Should be set to `true` only for non-compliant OCI registries which are missing the `/v2` prefix, and the API root endpoint is defined in the host URL path.<br />If not set, the `override_path` field defaults to `false` in containerd registry configuration.</p>
</td>
</tr>

</tbody>
</table>


<h3 id="mirrorhostcapability">MirrorHostCapability
</h3>
<p><em>Underlying type: string</em></p>


<p>
(<em>Appears on:</em><a href="#mirrorhost">MirrorHost</a>)
</p>

<p>
MirrorHostCapability represents a mirror host capability.
</p>


