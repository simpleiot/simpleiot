module Utils.NodeTree exposing
    ( anchorOf
    , applyEdgePoints
    , applyPoints
    , clearLoading
    , expandedIDs
    , findByFeID
    , finish
    , isTombstone
    , mergeForest
    , nodeView
    , replaceAnchor
    , replaceChild
    , replaceChildren
    , setExpanded
    , watchSubjects
    )

{-| The node tree the home page shows: one tree per anchor (a group the
user belongs to), fetched one subtree at a time and kept current from
live points. Everything here is pure so it can be tested.

A reply to a fetch is a flat list of nodes; the parent field of each says
where it goes. `depth` is how many levels below the requested nodes the
reply reached: nodes at that level keep whatever children the tree already
had for them, since the reply did not look.

-}

import Api.Node as Node exposing (Node, NodeView)
import Api.Point as Point exposing (Point)
import Dict
import List.Extra
import Tree exposing (Tree)



-- BUILDING


{-| nodeView wraps a node fetched under an anchor.
-}
nodeView : String -> Node -> NodeView
nodeView anchor node =
    { node = node
    , feID = 0
    , parentID = node.parent
    , anchor = anchor
    , hasChildren = False
    , expDetail = False
    , expChildren = False
    , loading = False
    , mod = False
    }


{-| buildForest builds the subtrees of the nodes in the list whose parent
is the one given, taking their descendants from the same list.
-}
buildForest : String -> String -> List Node -> List (Tree NodeView)
buildForest anchor parent nodes =
    nodes
        |> List.filter (\n -> n.parent == parent)
        |> List.map (buildTree anchor nodes [])


buildTree : String -> List Node -> List String -> Node -> Tree NodeView
buildTree anchor nodes path node =
    let
        children =
            if List.member node.id path then
                []

            else
                nodes
                    |> List.filter (\n -> n.parent == node.id)
                    |> List.map (buildTree anchor nodes (node.id :: path))
    in
    Tree.tree (nodeView anchor node) children



-- MERGING


{-| mergeForest merges freshly fetched subtrees into the ones already
there, matching by node ID. Expansion state is kept, nodes that are gone
are dropped, and at the depth the reply did not reach the old children
stay.
-}
mergeForest : Int -> List (Tree NodeView) -> List (Tree NodeView) -> List (Tree NodeView)
mergeForest depth old new =
    List.map
        (\n ->
            case List.Extra.find (\o -> (Tree.label o).node.id == (Tree.label n).node.id) old of
                Just o ->
                    mergeTree depth o n

                Nothing ->
                    n
        )
        new


mergeTree : Int -> Tree NodeView -> Tree NodeView -> Tree NodeView
mergeTree depth old new =
    let
        o =
            Tree.label old

        n =
            Tree.label new

        label =
            { n | expChildren = o.expChildren, expDetail = o.expDetail }

        children =
            if depth <= 0 then
                Tree.children old

            else
                mergeForest (depth - 1) (Tree.children old) (Tree.children new)
    in
    Tree.tree label children


{-| replaceAnchor puts the tree fetched for an anchor (a request for
`nodes.all.<anchor>`) in place of the one already there, or adds it.
-}
replaceAnchor : String -> Int -> List Node -> List (Tree NodeView) -> List (Tree NodeView)
replaceAnchor anchor depth nodes forest =
    let
        ( mine, others ) =
            List.partition (\t -> (Tree.label t).anchor == anchor) forest
    in
    case List.Extra.find (\n -> n.id == anchor) nodes of
        Nothing ->
            others

        Just top ->
            let
                new =
                    Tree.tree
                        (nodeView anchor { top | parent = "root" })
                        (buildForest anchor anchor nodes)
            in
            others ++ mergeForest depth mine [ new ]


{-| replaceChildren puts the children fetched for a node (a request for
`nodes.<parent>.all`) under every copy of that node in the anchor's tree,
and clears its loading flag.
-}
replaceChildren : String -> String -> Int -> List Node -> List (Tree NodeView) -> List (Tree NodeView)
replaceChildren anchor parentID depth nodes forest =
    let
        new =
            buildForest anchor parentID nodes
    in
    inAnchor anchor
        (replaceIn parentID
            (\t ->
                Tree.tree (loaded (Tree.label t)) (mergeForest depth (Tree.children t) new)
            )
        )
        forest


{-| replaceChild puts one child fetched for a node (a request for
`nodes.<parent>.<id>`) under every copy of that node in the anchor's
tree, replacing the copy that was there or adding it.
-}
replaceChild : String -> String -> String -> Int -> List Node -> List (Tree NodeView) -> List (Tree NodeView)
replaceChild anchor parentID childID depth nodes forest =
    case buildForest anchor parentID nodes |> List.filter (\t -> (Tree.label t).node.id == childID) |> List.head of
        Nothing ->
            forest

        Just new ->
            inAnchor anchor
                (replaceIn parentID
                    (\t ->
                        let
                            children =
                                Tree.children t
                        in
                        Tree.tree (loaded (Tree.label t)) <|
                            if List.any (\c -> (Tree.label c).node.id == childID) children then
                                List.map
                                    (\c ->
                                        if (Tree.label c).node.id == childID then
                                            mergeTree depth c new

                                        else
                                            c
                                    )
                                    children

                            else
                                children ++ [ new ]
                    )
                )
                forest


loaded : NodeView -> NodeView
loaded n =
    { n | loading = False }


inAnchor : String -> (Tree NodeView -> Tree NodeView) -> List (Tree NodeView) -> List (Tree NodeView)
inAnchor anchor f forest =
    List.map
        (\t ->
            if (Tree.label t).anchor == anchor then
                f t

            else
                t
        )
        forest


{-| replaceIn applies a function to every subtree whose root has the ID.
-}
replaceIn : String -> (Tree NodeView -> Tree NodeView) -> Tree NodeView -> Tree NodeView
replaceIn id f tree =
    let
        l =
            Tree.label tree
    in
    if l.node.id == id then
        f tree

    else
        Tree.tree l (List.map (replaceIn id f) (Tree.children tree))



-- LIVE POINTS


{-| applyPoints merges points into every copy of a node.
-}
applyPoints : String -> List Point -> List (Tree NodeView) -> List (Tree NodeView)
applyPoints nodeID points forest =
    List.map
        (Tree.map
            (\n ->
                if n.node.id == nodeID then
                    let
                        node =
                            n.node
                    in
                    { n | node = { node | points = Point.updatePoints node.points points } }

                else
                    n
            )
        )
        forest


{-| applyEdgePoints merges points into the edge between a node and its
parent wherever the tree has it. Where the tree has the parent but not the
child, it reports the parent to fetch the child under, once per anchor,
along with whether the parent is expanded.
-}
applyEdgePoints :
    String
    -> String
    -> List Point
    -> List (Tree NodeView)
    -> ( List (Tree NodeView), List { anchor : String, parentID : String, expanded : Bool } )
applyEdgePoints nodeID parentID points forest =
    let
        updated =
            List.map
                (Tree.map
                    (\n ->
                        if n.node.id == nodeID && n.node.parent == parentID then
                            let
                                node =
                                    n.node
                            in
                            { n | node = { node | edgePoints = Point.updatePoints node.edgePoints points } }

                        else
                            n
                    )
                )
                forest

        missing =
            forest
                |> List.filterMap
                    (\t ->
                        let
                            parents =
                                Tree.flatten t |> List.filter (\n -> n.node.id == parentID)

                            hasChild =
                                Tree.flatten t |> List.any (\n -> n.node.id == nodeID && n.node.parent == parentID)
                        in
                        if List.isEmpty parents || hasChild then
                            Nothing

                        else
                            Just
                                { anchor = (Tree.label t).anchor
                                , parentID = parentID
                                , expanded = List.any .expChildren parents
                                }
                    )
    in
    ( updated, missing )



-- STATE


setExpanded : Int -> Bool -> List (Tree NodeView) -> List (Tree NodeView)
setExpanded feID expanded forest =
    List.map
        (Tree.map
            (\n ->
                if n.feID == feID then
                    { n | expChildren = expanded, loading = expanded }

                else
                    n
            )
        )
        forest


clearLoading : List (Tree NodeView) -> List (Tree NodeView)
clearLoading =
    List.map (Tree.map loaded)


{-| anchorOf is the anchor a node was fetched under, which a request
about it has to name. A node in two groups may have two; the first is as
good as any.
-}
anchorOf : String -> List (Tree NodeView) -> Maybe String
anchorOf id forest =
    forest
        |> List.concatMap Tree.flatten
        |> List.Extra.find (\n -> n.node.id == id)
        |> Maybe.map .anchor


findByFeID : Int -> List (Tree NodeView) -> Maybe NodeView
findByFeID feID forest =
    forest
        |> List.concatMap Tree.flatten
        |> List.Extra.find (\n -> n.feID == feID)


{-| expandedIDs lists every expanded node that is on screen, as the
anchor and ID to refetch its children with.
-}
expandedIDs : List (Tree NodeView) -> List { anchor : String, id : String }
expandedIDs forest =
    List.concatMap
        (\t ->
            visible t
                |> List.filter .expChildren
                |> List.map (\n -> { anchor = n.anchor, id = n.node.id })
        )
        forest
        |> List.Extra.uniqueBy (\r -> r.anchor ++ "/" ++ r.id)


{-| visible lists the nodes a tree shows: the root, and the live children
of every expanded node below it.
-}
visible : Tree NodeView -> List NodeView
visible tree =
    let
        l =
            Tree.label tree
    in
    l
        :: (if l.expChildren then
                Tree.children tree
                    |> List.filter (\c -> not (isTombstone (Tree.label c).node))
                    |> List.concatMap visible

            else
                []
           )


{-| watchSubjects is what the page needs to hear about: the points of
every node on screen, and the edges of its children, so a new, deleted, or
re-roled child is noticed.
-}
watchSubjects : List (Tree NodeView) -> List String
watchSubjects forest =
    forest
        |> List.concatMap
            (\t ->
                visible t
                    |> List.concatMap
                        (\n ->
                            [ "up." ++ n.anchor ++ "." ++ n.node.id ++ ".*.*"
                            , "up." ++ n.anchor ++ ".*." ++ n.node.id ++ ".*.*"
                            ]
                        )
            )
        |> List.Extra.unique
        |> List.sort


isTombstone : Node -> Bool
isTombstone node =
    Point.getBool node.edgePoints Point.typeTombstone ""



-- FINISHING


{-| finish brings a changed forest back to what the view expects: each
node knows whether it has live children, siblings are sorted, and every
node has a front-end ID.
-}
finish : List (Tree NodeView) -> List (Tree NodeView)
finish forest =
    forest
        |> List.map (populateHasChildren "")
        |> sortNodeTrees
        |> populateFeID



-- FeID stands for front-end ID. This is required because we may
-- have some duplicate nodes in the data set, so we simply give each
-- one a unique ID while we are working with them in the frontend


populateFeID : List (Tree NodeView) -> List (Tree NodeView)
populateFeID trees =
    List.indexedMap
        (\i nodes ->
            Tree.indexedMap
                (\j n ->
                    { n | feID = i * 10000 + j }
                )
                nodes
        )
        trees


populateHasChildren : String -> Tree NodeView -> Tree NodeView
populateHasChildren parentID tree =
    let
        children =
            Tree.children tree

        hasChildren =
            List.any (\child -> not (isTombstone (Tree.label child).node)) children

        label =
            Tree.label tree

        node =
            { label
                | hasChildren = hasChildren
                , parentID = parentID
            }
    in
    tree
        |> Tree.replaceLabel node
        |> Tree.replaceChildren
            (List.map
                (\c -> populateHasChildren node.node.id c)
                children
            )


sortNodeTrees : List (Tree NodeView) -> List (Tree NodeView)
sortNodeTrees trees =
    List.sortWith nodeSort trees |> List.map sortNodeTree



-- sortNodeTree recursively sorts the children of the nodes
-- sort by type and then description


sortNodeTree : Tree NodeView -> Tree NodeView
sortNodeTree nodes =
    let
        children =
            Tree.children nodes

        childrenSorted =
            List.sortWith nodeSort children
    in
    Tree.tree (Tree.label nodes) (List.map sortNodeTree childrenSorted)



-- nodeCustomSortRules struct determines how we sort nodes in the UI


nodeCustomSortRules : Dict.Dict String String
nodeCustomSortRules =
    Dict.fromList
        [ ( Node.typeDevice, "A" )
        , ( Node.typeUser, "B" )
        , ( Node.typeGroup, "C" )
        , ( Node.typeModbus, "D" )
        , ( Node.typeRule, "E" )
        , ( Node.typeSignalGenerator, "F" )
        , ( Node.typeOneWire, "G" )
        , ( Node.typeGpio, "G1" )
        , ( Node.typeIio, "G2" )
        , ( Node.typeCanBus, "H" )
        , ( Node.typeSerialDev, "I" )
        , ( Node.typeMsgService, "J" )
        , ( Node.typeFile, "K" )
        , ( Node.typeVariable, "L" )
        , ( Node.typeDb, "M" )
        , ( Node.typeMetrics, "N" )
        , ( Node.typeParticle, "O" )
        , ( Node.typeShelly, "P" )
        , ( Node.typeShellyIO, "Q" )
        , ( Node.typeNetworkManager, "R" )
        , ( Node.typeNTP, "S" )
        , ( Node.typeUpdate, "T" )
        , ( Node.typeBrowser, "U" )
        , ( Node.typeGps, "V" )
        , ( Node.typeMqtt, "W" )
        , ( Node.typeMqttDevice, "W1" )
        , ( Node.typeSparkplugGroup, "X" )
        , ( Node.typeSparkplugNode, "Y" )
        , ( Node.typeSparkplugDevice, "Z" )

        -- rule subnodes
        , ( Node.typeCondition, "A" )
        , ( Node.typeAction, "B" )
        , ( Node.typeActionInactive, "C" )
        , ( Node.typeNetworkManagerDevice, "D" )
        , ( Node.typeNetworkManagerConn, "E" )
        ]


nodeCustomSort : String -> String
nodeCustomSort t =
    case Dict.get t nodeCustomSortRules of
        Just s ->
            s

        Nothing ->
            t


nodeSort : Tree NodeView -> Tree NodeView -> Order
nodeSort a b =
    let
        aNode =
            Tree.label a

        bNode =
            Tree.label b

        aType =
            nodeCustomSort aNode.node.typ

        bType =
            nodeCustomSort bNode.node.typ
    in
    if aType /= bType then
        compare aType bType

    else
        let
            aDesc =
                String.toLower <| Point.getBestDesc aNode.node.points

            bDesc =
                String.toLower <| Point.getBestDesc bNode.node.points
        in
        if aDesc /= bDesc then
            compare aDesc bDesc

        else
            let
                aIndex =
                    Point.getValue aNode.node.points Point.typeIndex ""

                bIndex =
                    Point.getValue bNode.node.points Point.typeIndex ""
            in
            if aIndex /= bIndex then
                compare aIndex bIndex

            else
                let
                    aID =
                        Point.getText aNode.node.points Point.typeID ""

                    bID =
                        Point.getText bNode.node.points Point.typeID ""
                in
                compare aID bID
