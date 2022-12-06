module Components.NodeFile exposing (view)

import Api.Point as Point
import Components.NodeOptions exposing (NodeOptions, oToInputO)
import Element exposing (..)
import Element.Background as Background
import Element.Border as Border
import Element.Font as Font
import File.Select as Select
import Round
import UI.Form as Form
import UI.Icon as Icon
import UI.NodeInputs as NodeInputs
import UI.Style as Style exposing (colors)
import UI.ViewIf exposing (viewIf)


view : NodeOptions msg -> Element msg
view o =
    let
        labelWidth =
            150

        opts =
            oToInputO o labelWidth

        textInput =
            NodeInputs.nodeTextInput opts ""

        desc =
            Point.getText o.node.points Point.typeDescription ""

        name =
            Point.getText o.node.points Point.typeName ""
    in
    column
        [ width fill
        , Border.widthEach { top = 2, bottom = 0, left = 0, right = 0 }
        , Border.color Style.colors.black
        , spacing 6
        ]
    <|
        wrappedRow [ spacing 10 ]
            [ Icon.file
            , text <|
                desc
                    ++ " ("
                    ++ name
                    ++ ")"
            ]
            :: (if o.expDetail then
                    [ textInput Point.typeDescription "Description" ""
                    , textInput Point.typeName "Name" ""
                    , Form.buttonRow
                        [ Form.button
                            { label = "upload"
                            , color = colors.blue
                            , onPress = o.onUploadFile
                            }
                        ]
                    ]

                else
                    []
               )
