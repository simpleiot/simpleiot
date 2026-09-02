module NodeTree exposing (all)

import Api.Node exposing (Node, NodeView)
import Api.Point as Point exposing (Point)
import Expect
import Test exposing (..)
import Time
import Tree exposing (Tree)
import Utils.NodeTree as NodeTree


node : String -> String -> String -> Node
node id typ parent =
    { id = id
    , typ = typ
    , hash = 0
    , parent = parent
    , points = [ Point.newText Point.typeDescription "0" id ]
    , edgePoints = [ tombstone 0 ]
    }


tombstone : Float -> Point
tombstone v =
    Point Point.typeTombstone "0" (Time.millisToPosix 0) 1 v "" 0


view : String -> Node -> NodeView
view =
    NodeTree.nodeView


tree : String -> Node -> List (Tree NodeView) -> Tree NodeView
tree anchor n children =
    Tree.tree (view anchor n) children


expand : Tree NodeView -> Tree NodeView
expand t =
    Tree.replaceLabel
        (let
            l =
                Tree.label t
         in
         { l | expChildren = True }
        )
        t


labels : List (Tree NodeView) -> List String
labels =
    List.map (Tree.label >> .node >> .id)


all : Test
all =
    describe "The NodeTree module"
        [ test "mergeForest keeps expansion and the levels the reply did not reach" <|
            \_ ->
                let
                    old =
                        [ expand <|
                            tree "G"
                                (node "a" "group" "G")
                                [ tree "G" (node "a1" "variable" "a") [ tree "G" (node "a1x" "variable" "a1") [] ]
                                , tree "G" (node "stale" "variable" "a") []
                                ]
                        ]

                    new =
                        [ tree "G"
                            (node "a" "group" "G")
                            [ tree "G" (node "a1" "variable" "a") []
                            , tree "G" (node "a2" "variable" "a") []
                            ]
                        ]

                    merged =
                        NodeTree.mergeForest 1 old new
                in
                Expect.all
                    [ \m -> Expect.equal True (List.head m |> Maybe.map (Tree.label >> .expChildren) |> Maybe.withDefault False)
                    , \m -> Expect.equal [ "a1", "a2" ] (List.head m |> Maybe.map Tree.children |> Maybe.withDefault [] |> labels)
                    , \m ->
                        -- a1 sat at the depth the reply stopped at, so its
                        -- children stay
                        Expect.equal [ "a1x" ]
                            (List.head m
                                |> Maybe.map Tree.children
                                |> Maybe.andThen List.head
                                |> Maybe.map Tree.children
                                |> Maybe.withDefault []
                                |> labels
                            )
                    ]
                    merged
        , test "replaceChildren fills in a fetched subtree and clears loading" <|
            \_ ->
                let
                    forest =
                        NodeTree.setExpanded 0
                            True
                            (NodeTree.finish [ tree "G" (node "G" "group" "root") [] ])

                    reply =
                        [ node "c1" "variable" "G", node "c2" "group" "G", node "c2a" "variable" "c2" ]

                    result =
                        NodeTree.finish (NodeTree.replaceChildren "G" "G" 1 reply forest)

                    top =
                        List.head result
                in
                Expect.all
                    [ \_ -> Expect.equal (Just False) (Maybe.map (Tree.label >> .loading) top)
                    , \_ -> Expect.equal [ "c2", "c1" ] (Maybe.map Tree.children top |> Maybe.withDefault [] |> labels)
                    , \_ ->
                        Expect.equal [ True, False ]
                            (Maybe.map Tree.children top |> Maybe.withDefault [] |> List.map (Tree.label >> .hasChildren))
                    ]
                    ()
        , test "watchSubjects covers the nodes on screen" <|
            \_ ->
                let
                    forest =
                        [ expand <|
                            tree "G"
                                (node "G" "group" "root")
                                [ tree "G" (node "c1" "variable" "G") [ tree "G" (node "hidden" "variable" "c1") [] ]
                                , tree "G" { id = "gone", typ = "variable", hash = 0, parent = "G", points = [], edgePoints = [ tombstone 1 ] } []
                                ]
                        ]
                in
                Expect.equal
                    [ "up.G.*.G.*.*"
                    , "up.G.*.c1.*.*"
                    , "up.G.G.*.*"
                    , "up.G.c1.*.*"
                    ]
                    (NodeTree.watchSubjects forest)
        , test "a tombstone edge point hides a node" <|
            \_ ->
                let
                    forest =
                        [ tree "G" (node "G" "group" "root") [ tree "G" (node "c1" "variable" "G") [] ] ]

                    ( result, missing ) =
                        NodeTree.applyEdgePoints "c1" "G" [ tombstone 1 ] forest

                    child =
                        List.head result |> Maybe.map Tree.children |> Maybe.andThen List.head |> Maybe.map Tree.label
                in
                Expect.all
                    [ \_ -> Expect.equal (Just True) (Maybe.map (.node >> NodeTree.isTombstone) child)
                    , \_ -> Expect.equal [] missing
                    ]
                    ()
        , test "an edge for a node the tree lacks names the parent to fetch under" <|
            \_ ->
                let
                    forest =
                        [ expand <| tree "G" (node "G" "group" "root") [] ]

                    ( _, missing ) =
                        NodeTree.applyEdgePoints "new" "G" [ tombstone 0 ] forest
                in
                Expect.equal [ { anchor = "G", parentID = "G", expanded = True } ] missing
        ]
