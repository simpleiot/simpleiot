module Components.NodeRaw exposing (view)

import Api.Point as Point exposing (Point)
import Components.NodeOptions exposing (NodeOptions, oToInputO)
import Element exposing (..)
import Element.Background as Background
import Element.Border as Border
import Element.Font as Font
import UI.Button as Button
import UI.NodeInputs as NodeInputs
import UI.Style as Style


view : NodeOptions msg -> Element msg
view o =
    column
        [ width fill
        , Border.widthEach { top = 2, bottom = 0, left = 0, right = 0 }
        , Border.color Style.colors.black
        , spacing 6
        ]
    <|
        let
            description =
                Point.getText o.node.points Point.typeDescription ""
        in
        wrappedRow [ spacing 10 ]
            [ Element.el [ Background.color Style.colors.yellow ] <| Element.text <| "Node type: " ++ o.node.typ
            , text description
            ]
            :: (if o.expDetail then
                    let
                        opts =
                            oToInputO o 0
                    in
                    [ viewPoints opts
                    ]

                else
                    []
               )


viewPoints : NodeInputs.NodeInputOptions msg -> Element msg
viewPoints o =
    table [ padding 7 ]
        { data = o.node.points |> Point.filterTombstone |> List.sortWith Point.sort |> List.map renderPoint
        , columns =
            let
                cell =
                    el [ paddingXY 15 5, Border.width 0 ]
            in
            [ { header = cell <| el [ Font.bold, centerX ] <| text "Point"
              , width = fill
              , view = \m -> cell <| text m.desc
              }
            , { header = cell <| el [ Font.bold, centerX ] <| text "Value"
              , width = fill
              , view = \m -> cell <| el [ alignRight ] <| NodeInputs.nodeNumberInput o m.p.key m.p.typ ""
              }
            , { header = cell <| el [ Font.bold, centerX ] <| text "Text"
              , width = fill
              , view =
                    \m ->
                        cell <|
                            el [ alignRight ] <|
                                NodeInputs.nodeTextInput o m.p.key m.p.typ "" ""
              }
            , { header = cell <| el [ Font.bold, centerX ] <| text ""
              , width = fill
              , view =
                    \m ->
                        Button.x <|
                            o.onEditNodePoint [ Point m.p.typ m.p.key o.now 0 "" 1 ]
              }
            ]
        }


renderPoint : Point -> { desc : String, value : Float, text : String, p : Point }
renderPoint p =
    let
        key =
            p.key

        value =
            p.value

        text =
            p.text
    in
    { desc = p.typ ++ ":" ++ key, value = value, text = text, p = p }
