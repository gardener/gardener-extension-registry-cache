<p>Packages:</p>
<ul>
<li>
<a href="#mirror.extensions.gardener.cloud%2fv1alpha1">mirror.extensions.gardener.cloud/v1alpha1</a>
</li>
</ul>
<h2 id="mirror.extensions.gardener.cloud/v1alpha1">mirror.extensions.gardener.cloud/v1alpha1</h2>
<p>
<p>Package v1alpha1 contains the Registry Cache Service extension configuration.</p>
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
<p>Upstream is the remote registry host to mirror.</p>
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
Defaults to [&ldquo;pull&rdquo;].</p>
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
