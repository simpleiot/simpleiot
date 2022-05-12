module Components.NodeOneWireIO exposing (view)

import Api.Point as Point
import Components.NodeOptions exposing (NodeOptions, oToInputO)
import Element exposing (..)
import Element.Border as Border
import Round
import UI.Icon as Icon
import UI.NodeInputs as NodeInputs
import UI.Style exposing (colors)


view : NodeOptions msg -> Element msg
view o =
    let
        labelWidth =
            150

        opts =
            oToInputO o labelWidth

        textInput =
            NodeInputs.nodeTextInput opts ""

        value =
            Point.getValue o.node.points Point.typeValue ""

        valueText =
            String.fromFloat (Round.roundNum 2 value) ++ "°C"

        id =
            Point.getText o.node.points Point.typeID ""
    in
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
            , el [ paddingXY 7 0 ] <|
                text <|
                    valueText
            ]
            :: (if o.expDetail then
                    [ el [ paddingEach { top = 0, right = 0, bottom = 0, left = 70 } ] <|
                        text <|
                            "ID: "
                                ++ id
                    , textInput Point.typeDescription "Description" ""
                    ]

                else
                    []
               )
