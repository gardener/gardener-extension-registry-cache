<p>Packages:</p>
<ul>
<li>
<a href="#spegel.extensions.gardener.cloud%2fv1alpha1">spegel.extensions.gardener.cloud/v1alpha1</a>
</li>
</ul>

<h2 id="spegel.extensions.gardener.cloud/v1alpha1">spegel.extensions.gardener.cloud/v1alpha1</h2>
<p>

</p>

<h3 id="spegelconfig">SpegelConfig
</h3>


<p>
SpegelConfig contains information about the Spegel listening addresses of each Node.
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
<code>registryPort</code></br>
<em>
integer
</em>
</td>
<td>
<em>(Optional)</em>
<p>RegistryPort is the port that serves the OCI registry on each Node.<br />`registryPort` should be a valid port number (1-65535, inclusive).<br />Defaults to 15500.</p>
</td>
</tr>
<tr>
<td>
<code>routerPort</code></br>
<em>
integer
</em>
</td>
<td>
<em>(Optional)</em>
<p>RouterPort is the port for P2P router on each Node.<br />`routerPort` should be a valid port number (1-65535, inclusive).<br />Defaults to 15501.</p>
</td>
</tr>
<tr>
<td>
<code>metricsPort</code></br>
<em>
integer
</em>
</td>
<td>
<em>(Optional)</em>
<p>MetricsPort is the metrics port on each Node.<br />`metricsPort` should be a valid port number (1-65535, inclusive).<br />Defaults to 19090.</p>
</td>
</tr>
<tr>
<td>
<code>resolveTags</code></br>
<em>
boolean
</em>
</td>
<td>
<em>(Optional)</em>
<p>ResolveTags when true Spegel will resolve tags to digests.<br />Defaults to true.</p>
</td>
</tr>

</tbody>
</table>


<h3 id="spegelstatus">SpegelStatus
</h3>


<p>
SpegelStatus contains information about Spegel client TLS secrets.
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
<code>caSecretName</code></br>
<em>
string
</em>
</td>
<td>
<p>CASecretName is the name of the CA bundle secret.</p>
</td>
</tr>
<tr>
<td>
<code>clientTLSSecretName</code></br>
<em>
string
</em>
</td>
<td>
<p>ClientTLSSecretName is the name of the Spegel client TLS secret.</p>
</td>
</tr>

</tbody>
</table>


