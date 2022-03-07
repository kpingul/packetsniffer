//Global variables
const API_RECORDS = "http://127.0.0.1:8090/api/records"
const PROTOCOLS = {
        IP: "IP",
        HTTP: "HTTP",
        DNS: "DNS"
}

//cytoscape configuration
var cytoConfig = {
        nodes: [],
        nodesObj: {},
        nodes: [],
        edgesObj: {},
        edges: []
}


//create XMLHttpRequest object
const xhr = new XMLHttpRequest()

//open a get request with the remote server URL
xhr.open("GET", API_RECORDS)

//send the Http request
xhr.send()

//EVENT HANDLERS
//triggered when the response is completed
xhr.onload = function() {
        if (xhr.status === 200) {

                //parse JSON data
                data = JSON.parse(xhr.responseText)
                console.log(data)

                if ( data.length > 0 ) {


                        //creates a nodes object to
                        //keep track of duplication
                        //**should move to server side
                        //to prevent performance issues
                        data.forEach( (node, idx) => {
                                if ( !cytoConfig.nodesObj.hasOwnProperty(node.SrcIP) ) {
                                        cytoConfig.nodesObj[node.SrcIP] = createNode(node.SrcIP, PROTOCOLS.IP, node.SrcIP, "", node.SrcIP, node.DstIP, "circle")

                                }
                                if ( !cytoConfig.nodesObj.hasOwnProperty(node.DstIP) ) {
                                        cytoConfig.nodesObj[node.DstIP] = createNode(node.DstIP, PROTOCOLS.IP, node.DstIP, "", node.SrcIP, node.DstIP, "circle")

                                }
                                if ( node.HTTPHeader && node.HTTPHeader.Host  ) {
                                        cytoConfig.nodesObj[node.HTTPHeader.Host] = createNode(node.HTTPHeader.Host, PROTOCOLS.HTTP, node.HTTPHeader.Host, node.HTTPHeader.Type, node.SrcIP, node.HTTPHeader.Host, "circle")
                                } 
                                if ( node.DNS && node.DNS.Domain  ) {
                                        cytoConfig.nodesObj[node.DNS.Domain] = createNode(node.DNS.Domain, PROTOCOLS.DNS, node.DNS.Domain, node.DNS.Type, node.SrcIP, node.DNS.Domain, "circle")
                                }

                        })
                        
                        //store all our unique nodes into the nodes variable                                                
                        Object.keys(cytoConfig.nodesObj).forEach( (key, val) => {
                                cytoConfig.nodes.push(cytoConfig.nodesObj[key])
                        })


                        //creates our edges/links 
                        cytoConfig.nodes.forEach( (node,idx) => {
                                console.log(node)
                                if ( node.data.type == PROTOCOLS.IP) {

                                        cytoConfig.edges.push({
                                                data: {
                                                        id: node.data.srcIP + "-" + node.data.dstIP,
                                                        source: node.data.srcIP,
                                                        target: node.data.dstIP,
                                                        label: ""
                                                }
                                        })
                                } 
                                if ( node.data.type == PROTOCOLS.HTTP) {
                                        cytoConfig.edges.push({
                                                data: {
                                                        id: node.data.srcIP + "-" + node.data.dstIP,
                                                        source: node.data.srcIP,
                                                        target: node.data.dstIP,
                                                        label: node.data.linkLabel
                                                }
                                        })
                                }  
                                if ( node.data.type == PROTOCOLS.DNS) {
                                        cytoConfig.edges.push({
                                                data: {
                                                        id: node.data.srcIP + "-" + node.data.dstIP,
                                                        source: node.data.srcIP,
                                                        target: node.data.dstIP,
                                                        label: node.data.linkLabel
                                                }
                                        })
                                }
                        })

                        //run graph
                        runCytoScape()
                        
                }
        } else if (xhr.status === 404) {
                console.log("No records found")
        }
}

//triggered when a network-level error occurs with the request
xhr.onerror = function() {
        console.log("Network error occurred")
}


function createNode ( id, type, label, linkLabel, srcIP, dstIP, shape ) {
        return {
                data: {
                        id: id,
                        type: type,
                        label: label,
                        linkLabel: linkLabel,
                        srcIP: srcIP,
                        dstIP: dstIP,
                        shape: shape,
                }
        }
}
  
function runCytoScape() {
        var cy = cytoscape({
                container: document.getElementById('cy'),
                boxSelectionEnabled: false,
                autounselectify: true,
                style:  cytoscape.stylesheet()
                        .selector('node')
                        .style({
                                'content': 'data(label)',
                                'width': '20',
                                'height': '20',
                                // 'shape': 'data(shape)',
                                'font-size': '11',
                                "text-valign": "top",
                                "text-halign": "center"
                        })
                        .selector('edge')
                        .style({
                                'content': 'data(label)',
                                'font-size': '8',
                                'curve-style': 'bezier',
                                'target-arrow-shape': 'triangle',
                                'width': 2,
                                'line-color': '#ddd',
                                'target-arrow-color': '#ddd'
                        })
                        .selector('.highlighted')
                        .style({
                                'background-color': '#D2042D',
                                'line-color': '#B0C4DE ',
                                'target-arrow-color': '#B0C4DE ',
                                'transition-property': 'background-color, line-color, target-arrow-color',
                                'transition-duration': '0.5s'
                        }),
                elements: {
                        nodes: cytoConfig.nodes,
                        edges: cytoConfig.edges
                },

                layout: {
                        name: 'breadthfirst',
                        directed: true,
                        padding: 10
                }
        });


        //find a way to get the root node and highlight children 
        //cytoConfig.nodes contains the nodes, which one is root/host?
        var bfs = cy.elements().bfs('#' + cytoConfig.nodes[0].data.id, function(){}, true);

        var i = 0;
        var highlightNextEle = function(){
                if( i < bfs.path.length ){
                        bfs.path[i].addClass('highlighted');
                        i++;
                        setTimeout(highlightNextEle, 1000);
                }
        };

        //execute highlight nodes
        highlightNextEle()
}