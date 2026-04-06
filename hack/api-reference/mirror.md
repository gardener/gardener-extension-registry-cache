<p>Packages:</p>
<ul>
<li>
<a href="#mirror.extensions.gardener.cloud%2fv1alpha1">mirror.extensions.gardener.cloud/v1alpha1</a>
</li>
</ul>
<h2 id="mirror.extensions.gardener.cloud/v1alpha1">mirror.extensions.gardener.cloud/v1alpha1</h2>
<p>
<p>Package v1alpha1 is a version of the API.</p>
</p>
Resource Types:
<ul></ul>
<h3 id="mirror.extensions.gardener.cloud/v1alpha1.MirrorConfig">MirrorConfig
</h3>
<p>
<p>MirrorConfig contains information about registry mirrors to configure.</p>
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
<a href="#mirror.extensions.gardener.cloud/v1alpha1.MirrorConfiguration">
[]MirrorConfiguration
</a>
</em>
</td>
<td>
<p>Mirrors is a slice of registry mirrors to configure.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="mirror.extensions.gardener.cloud/v1alpha1.MirrorConfiguration">MirrorConfiguration
</h3>
<p>
(<em>Appears on:</em>
<a href="#mirror.extensions.gardener.cloud/v1alpha1.MirrorConfig">MirrorConfig</a>)
</p>
<p>
<p>MirrorConfiguration represents a registry mirror.</p>
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
<p>Upstream is the remote registry host to mirror.
The value must be a valid DNS subdomain (RFC 1123) and optionally a port.</p>
</td>
</tr>
<tr>
<td>
<code>hosts</code></br>
<em>
<a href="#mirror.extensions.gardener.cloud/v1alpha1.MirrorHost">
[]MirrorHost
</a>
</em>
</td>
<td>
<p>Hosts are the mirror hosts to be used for the upstream.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="mirror.extensions.gardener.cloud/v1alpha1.MirrorHost">MirrorHost
</h3>
<p>
(<em>Appears on:</em>
<a href="#mirror.extensions.gardener.cloud/v1alpha1.MirrorConfiguration">MirrorConfiguration</a>)
</p>
<p>
<p>MirrorHost represents a mirror host.</p>
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
<a href="#mirror.extensions.gardener.cloud/v1alpha1.MirrorHostCapability">
[]MirrorHostCapability
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Capabilities are the operations a host is capable of performing.
This also represents the set of operations for which the mirror host may be trusted to perform.
The supported values are &ldquo;pull&rdquo; and &ldquo;resolve&rdquo;.
Defaults to [&ldquo;pull&rdquo;].</p>
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
<p>CABundleSecretReferenceName is the reference name for a Secret containing a PEM-encoded certificate authority bundle.
The CA bundle is used to verify the TLS certificate of the mirror host.
The referenced secret must be immutable and must have a data key <code>bundle.crt</code>.</p>
</td>
</tr>
<tr>
<td>
<code>overridePath</code></br>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
<p>OverridePath represents the <code>override_path</code> field in the hosts.toml file for containerd registry configuration.
See <a href="https://github.com/containerd/containerd/blob/v2.2.0/docs/hosts.md#override_path-field">https://github.com/containerd/containerd/blob/v2.2.0/docs/hosts.md#override_path-field</a>
Should be set to <code>true</code> only for non-compliant OCI registries which are missing the <code>/v2</code> prefix, and the API root endpoint is defined in the host URL path.
If not set, the <code>override_path</code> field defaults to <code>false</code> in containerd registry configuration.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="mirror.extensions.gardener.cloud/v1alpha1.MirrorHostCapability">MirrorHostCapability
(<code>string</code> alias)</p></h3>
<p>
(<em>Appears on:</em>
<a href="#mirror.extensions.gardener.cloud/v1alpha1.MirrorHost">MirrorHost</a>)
</p>
<p>
<p>MirrorHostCapability represents a mirror host capability.</p>
</p>
<hr/>
<p><em>
Generated with <a href="https://github.com/ahmetb/gen-crd-api-reference-docs">gen-crd-api-reference-docs</a>
</em></p>
