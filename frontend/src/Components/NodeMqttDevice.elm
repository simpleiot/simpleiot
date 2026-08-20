module Components.NodeMqttDevice exposing (view)

import Api.Point as Point
import Components.NodeOptions exposing (NodeOptions, oToInputO)
import Element exposing (..)
import Element.Border as Border
import Element.Font as Font
import Round
import UI.Icon as Icon
import UI.NodeInputs as NodeInputs
import UI.Style exposing (colors)
import UI.ViewIf exposing (viewIf)


view : NodeOptions msg -> Element msg
view o =
    column
        [ width fill
        , Border.widthEach { top = 2, bottom = 0, left = 0, right = 0 }
        , Border.color colors.black
        , spacing 6
        ]
    <|
        wrappedRow [ spacing 10 ]
            [ Icon.io
            , text <|
                Point.getText o.node.points Point.typeDescription ""
            ]
            :: (if o.expDetail then
                    let
                        labelWidth =
                            150

                        opts =
                            oToInputO o labelWidth

                        textInput =
                            NodeInputs.nodeTextInput opts "0"

                        identity =
                            Point.getText o.node.points Point.typeID ""

                        values =
                            o.node.points
                                |> List.sortWith Point.sort
                                |> Point.filterDeleted
                                |> filterConfigPoints
                    in
                    [ el [ paddingEach { top = 0, right = 0, bottom = 0, left = 70 } ] <|
                        text <|
                            "Topic level: "
                                ++ identity
                    , textInput Point.typeDescription "Description" ""
                    , NodeInputs.nodeKeyValueInput opts Point.typeTag "Tags" "Add Tag"
                    , viewIf (List.length values > 0) <| viewValues values
                    ]

                else
                    []
               )


{-| Point types this component renders on its own, above the value table.
Listing them again alongside the values the device publishes only makes the
values harder to find. The values themselves are `value` points, which
`Point.filterSpecialPoints` would remove, so they are filtered here instead.
-}
configPoints : List String
configPoints =
    [ Point.typeDescription
    , Point.typeID
    , Point.typeTag
    ]


filterConfigPoints : List Point.Point -> List Point.Point
filterConfigPoints points =
    List.filter (\p -> not <| List.member p.typ configPoints) points


viewValues : List Point.Point -> Element msg
viewValues pts =
    table [ padding 7 ]
        { data = List.map formatValue pts
        , columns =
            let
                cell =
                    el [ paddingXY 15 5, Border.width 1 ]
            in
            [ { header = cell <| el [ Font.bold, centerX ] <| text "Point"
              , width = fill
              , view = \m -> cell <| text m.desc
              }
            , { header = cell <| el [ Font.bold, centerX ] <| text "Value"
              , width = fill
              , view = \m -> cell <| el [ alignRight ] <| text m.value
              }
            ]
        }


{-| A value point is named by its key, which the client builds from the topic
levels and payload fields below the ones the schema names. A payload carrying
a single measurement has no key, so the point type names it.
-}
formatValue : Point.Point -> { desc : String, value : String }
formatValue p =
    if p.typ == Point.typeValue then
        { desc =
            if p.key == "" then
                p.typ

            else
                p.key
        , value =
            if p.text /= "" then
                p.text

            else
                Round.round 2 p.value
        }

    else
        Point.renderPoint2 p
