<p>Packages:</p>
<ul>
<li>
<a href="#registry.extensions.gardener.cloud%2fv1alpha3">registry.extensions.gardener.cloud/v1alpha3</a>
</li>
</ul>

<h2 id="registry.extensions.gardener.cloud/v1alpha3">registry.extensions.gardener.cloud/v1alpha3</h2>
<p>

</p>

<h3 id="garbagecollection">GarbageCollection
</h3>


<p>
(<em>Appears on:</em><a href="#registrycache">RegistryCache</a>)
</p>

<p>
GarbageCollection contains settings for the garbage collection of content from the cache.
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
<code>ttl</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.33/#duration-v1-meta">Duration</a>
</em>
</td>
<td>
<p>TTL is the time to live of a blob in the cache.<br />Set to 0s to disable the garbage collection.<br />Defaults to 168h (7 days).</p>
</td>
</tr>

</tbody>
</table>


<h3 id="http">HTTP
</h3>


<p>
(<em>Appears on:</em><a href="#registrycache">RegistryCache</a>)
</p>

<p>
HTTP contains settings for the HTTP server that hosts the registry cache.
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
<code>tls</code></br>
<em>
boolean
</em>
</td>
<td>
<p>TLS indicates whether TLS is enabled for the HTTP server of the registry cache.<br />Defaults to true.</p>
</td>
</tr>

</tbody>
</table>


<h3 id="highavailability">HighAvailability
</h3>


<p>
(<em>Appears on:</em><a href="#registrycache">RegistryCache</a>)
</p>

<p>
HighAvailability contains settings for high availability of the registry cache.
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
<code>enabled</code></br>
<em>
boolean
</em>
</td>
<td>
<p>Enabled defines if the registry cache is scaled with the high availability feature.<br />For more details, see https://github.com/gardener/gardener/blob/master/docs/development/high-availability-of-components.md#system-components.</p>
</td>
</tr>

</tbody>
</table>


<h3 id="proxy">Proxy
</h3>


<p>
(<em>Appears on:</em><a href="#registrycache">RegistryCache</a>)
</p>

<p>
Proxy contains settings for a proxy used in the registry cache.
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
<code>httpProxy</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>HTTPProxy field represents the proxy server for HTTP connections which is used by the registry cache.</p>
</td>
</tr>
<tr>
<td>
<code>httpsProxy</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>HTTPSProxy field represents the proxy server for HTTPS connections which is used by the registry cache.</p>
</td>
</tr>

</tbody>
</table>


<h3 id="registrycache">RegistryCache
</h3>


<p>
(<em>Appears on:</em><a href="#registryconfig">RegistryConfig</a>)
</p>

<p>
RegistryCache represents a registry cache to deploy.
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
<p>Upstream is the remote registry host to cache.<br />The value must be a valid DNS subdomain (RFC 1123) and optionally a port.</p>
</td>
</tr>
<tr>
<td>
<code>remoteURL</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>RemoteURL is the remote registry URL. The format must be `<scheme><host>[:<port>]` where<br />`<scheme>` is `https://` or `http://` and `<host>[:<port>]` corresponds to the Upstream<br />If defined, the value is set as `proxy.remoteurl` in the registry [configuration](https://github.com/distribution/distribution/blob/main/docs/content/recipes/mirror.md#configure-the-cache)<br />and in containerd configuration as `server` field in [hosts.toml](https://github.com/containerd/containerd/blob/main/docs/hosts.md#server-field) file.</p>
</td>
</tr>
<tr>
<td>
<code>volume</code></br>
<em>
<a href="#volume">Volume</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Volume contains settings for the registry cache volume.</p>
</td>
</tr>
<tr>
<td>
<code>garbageCollection</code></br>
<em>
<a href="#garbagecollection">GarbageCollection</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>GarbageCollection contains settings for the garbage collection of content from the cache.<br />Defaults to enabled garbage collection.</p>
</td>
</tr>
<tr>
<td>
<code>secretReferenceName</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>SecretReferenceName is the reference name for a Secret containing the upstream registry credentials.</p>
</td>
</tr>
<tr>
<td>
<code>proxy</code></br>
<em>
<a href="#proxy">Proxy</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Proxy contains settings for a proxy used in the registry cache.</p>
</td>
</tr>
<tr>
<td>
<code>http</code></br>
<em>
<a href="#http">HTTP</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>HTTP contains settings for the HTTP server that hosts the registry cache.</p>
</td>
</tr>
<tr>
<td>
<code>highAvailability</code></br>
<em>
<a href="#highavailability">HighAvailability</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>HighAvailability contains settings for high availability of the registry cache.</p>
</td>
</tr>
<tr>
<td>
<code>serviceNameSuffix</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>ServiceNameSuffix allows to customize the naming of the deployed service.<br />If not specified, the service suffix will be generated from the upstream.</p>
</td>
</tr>

</tbody>
</table>


<h3 id="registrycachestatus">RegistryCacheStatus
</h3>


<p>
(<em>Appears on:</em><a href="#registrystatus">RegistryStatus</a>)
</p>

<p>
RegistryCacheStatus represents a deployed registry cache.
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
<p>Upstream is the remote registry host (and optionally port).</p>
</td>
</tr>
<tr>
<td>
<code>endpoint</code></br>
<em>
string
</em>
</td>
<td>
<p>Endpoint is the registry cache endpoint.<br />Examples: "https://10.4.246.205:5000", "http://10.4.26.127:5000"</p>
</td>
</tr>
<tr>
<td>
<code>remoteURL</code></br>
<em>
string
</em>
</td>
<td>
<p>RemoteURL is the remote registry URL.</p>
</td>
</tr>

</tbody>
</table>


<h3 id="registryconfig">RegistryConfig
</h3>


<p>
RegistryConfig contains information about registry caches to deploy.
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
<code>caches</code></br>
<em>
<a href="#registrycache">RegistryCache</a> array
</em>
</td>
<td>
<p>Caches is a slice of registry caches to deploy.</p>
</td>
</tr>

</tbody>
</table>


<h3 id="registrystatus">RegistryStatus
</h3>


<p>
RegistryStatus contains information about deployed registry caches.
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
<em>(Optional)</em>
<p>CASecretName is the name of the CA bundle secret.<br />The field is nil when there is no registry cache that enables TLS for the HTTP server.</p>
</td>
</tr>
<tr>
<td>
<code>caches</code></br>
<em>
<a href="#registrycachestatus">RegistryCacheStatus</a> array
</em>
</td>
<td>
<p>Caches is a slice of deployed registry caches.</p>
</td>
</tr>

</tbody>
</table>


<h3 id="volume">Volume
</h3>


<p>
(<em>Appears on:</em><a href="#registrycache">RegistryCache</a>)
</p>

<p>
Volume contains settings for the registry cache volume.
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
<code>size</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.33/#quantity-resource-api">Quantity</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Size is the size of the registry cache volume.<br />Defaults to 10Gi.<br />This field is immutable.</p>
</td>
</tr>
<tr>
<td>
<code>storageClassName</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>StorageClassName is the name of the StorageClass used by the registry cache volume.<br />This field is immutable.</p>
</td>
</tr>

</tbody>
</table>


