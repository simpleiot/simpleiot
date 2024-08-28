module Components.NodeFile exposing (view)

import Api.Point as Point
import Components.NodeOptions exposing (NodeOptions, oToInputO)
import Element exposing (..)
import Element.Border as Border
import UI.Form as Form
import UI.Icon as Icon
import UI.NodeInputs as NodeInputs
import UI.Style as Style exposing (colors)


view : NodeOptions msg -> Element msg
view o =
    let
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
                    let
                        labelWidth =
                            150

                        opts =
                            oToInputO o labelWidth

                        textInput =
                            NodeInputs.nodeTextInput opts "0"

                        checkbox =
                            NodeInputs.nodeCheckboxInput opts "0"

                        binary =
                            Point.getBool o.node.points Point.typeBinary "0"

                        size =
                            Point.getValue o.node.points Point.typeSize "0"

                        data =
                            Point.getText o.node.points Point.typeData "0"
                    in
                    [ textInput Point.typeDescription "Description" ""
                    , checkbox "binary" "Binary"
                    , text <| " "
                    , text <| "     File name: " ++ name
                    , text <| "     File size: " ++ String.fromFloat size ++ " bytes"
                    , text <| "     Stored data len: " ++ String.fromInt (String.length data) ++ " bytes"
                    , Form.buttonRow
                        [ Form.button
                            { label = "Upload new file"
                            , color = colors.blue
                            , onPress = o.onUploadFile binary
                            }
                        ]
                    , NodeInputs.nodeKeyValueInput opts Point.typeTag "Tags" "Add Tag"
                    ]

                else
                    []
               )
